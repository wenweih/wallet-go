package blockchain

import (
	"os"
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"strings"
	"net/http"
	"path/filepath"
	"encoding/hex"
	"wallet-transition/pkg/db"
	"wallet-transition/pkg/util"
	"wallet-transition/pkg/configure"
	"github.com/btcsuite/btcd/btcjson"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcd/rpcclient"
	"github.com/btcsuite/btcd/wire"
	"github.com/btcsuite/btcutil"
	"github.com/manifoldco/promptui"
	"github.com/btcsuite/btcd/mempool"
	"github.com/btcsuite/btcutil/coinset"
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcutil/hdkeychain"
)

var btcWalletBackupPath = strings.Join([]string{configure.Config.BackupWalletPath, "btc.backup"}, "")

// BTCRPC bitcoin-core client alias
type BTCRPC struct {
	Client *rpcclient.Client
}

// NewbitcoinClient bitcoin rpc client
func NewbitcoinClient() *rpcclient.Client {
	connCfg := &rpcclient.ConnConfig {
		Host:         configure.Config.BTCNODEHOST,
		User:         configure.Config.BTCNODEUSR,
		Pass:         configure.Config.BTCNODEPASS,
		HTTPPostMode: configure.Config.BTCHTTPPostMode,
		DisableTLS:   configure.Config.BTCDisableTLS,
	}
	client, err := rpcclient.New(connCfg, nil)
	if err != nil {
		configure.Sugar.Fatal("bitcoind client err: ", err.Error())
	}
	return client
}

func NewOmnicoreClient() *rpcclient.Client {
	connCfg := &rpcclient.ConnConfig {
		Host:         configure.Config.OmniNODEHOST,
		User:         configure.Config.OmniNODEUSR,
		Pass:         configure.Config.OmniNODEPASS,
		HTTPPostMode: configure.Config.OmniHTTPPostMode,
		DisableTLS:   configure.Config.OmniDisableTLS,
	}
	client, err := rpcclient.New(connCfg, nil)
	if err != nil {
		configure.Sugar.Fatal("bitcoind client err: ", err.Error())
	}
	return client
}

// BtcUTXO utxo type
type BtcUTXO struct {
	Txid      string  `json:"txid"`
	Amount    float64 `json:"amount"`
	Height    int64   `json:"height"`
	VoutIndex uint32  `json:"voutindex"`
}

// DumpBTC dump wallet from node
func (btcClient *BTCRPC) DumpBTC(local bool) {
	oldWalletServerClient, err := util.NewServerClient(configure.Config.OldBTCWalletServerUser,
		configure.Config.OldBTCWalletServerPass, configure.Config.OldBTCWalletServerHost)
	if err != nil {
		configure.Sugar.Fatal(err.Error())
	}
	if err = oldWalletServerClient.SftpClient.MkdirAll(filepath.Dir(configure.Config.BackupWalletPath)); err != nil {
		configure.Sugar.Fatal(err.Error())
	}

	// dump old wallet to old wallet server
	btcClient.DumpOldWallet(oldWalletServerClient)
	oldWalletServerClient.CopyRemoteFile2(btcWalletBackupPath, local)
}

// DumpOldWallet migrate old wallet from node
func (btcClient *BTCRPC) DumpOldWallet(serverClient *util.ServerClient) {
	if _, err := btcClient.Client.DumpWallet(btcWalletBackupPath); err != nil {
		if strings.Contains(err.Error(), "already exists. If you are sure this is what you want") {
			prompt := promptui.Prompt {
				Label:     strings.Join([]string{"File: ", filepath.Base(configure.Config.BackupWalletPath),
					"backup wallet already exists, If you are sure this is what you want, move it out of the way first "}, ""),
				IsConfirm: true,
			}
			if _, err = prompt.Run(); err != nil {
				fmt.Println("Check the old backup wallet file in", configure.Config.BackupWalletPath, "in", serverClient.SSHClient.RemoteAddr().String())
				return
			}
			if err = serverClient.SftpClient.Remove(btcWalletBackupPath); err != nil {
				configure.Sugar.Fatal("Remove old backup wallet from old wallet server error: ", err.Error())
			}
			btcClient.DumpOldWallet(serverClient)
		} else {
			configure.Sugar.Fatalf("DumpWallet error: ", err.Error())
		}
	} else {
		configure.Sugar.Info("dump old btc wallet result: success")
	}
}

// ImportPrivateKey import private key from dump file
func (btcClient *BTCRPC) ImportPrivateKey()  {
	file, err := os.Open(strings.Join([]string{configure.Config.BackupWalletPath, "btc.backup_new"}, ""))
	if err != nil {
		configure.Sugar.Fatal("open dump wallet file error: ", err.Error())
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.Contains(line, "label=") && !strings.Contains(line, "script"){
			splitArr := strings.Split(line, " ")
			privateKey := splitArr[0]
			wif, err := btcutil.DecodeWIF(privateKey)
			if err != nil {
				configure.Sugar.Fatal("decode wif string error:", err.Error())
			}
			if result := btcClient.Client.ImportPrivKeyRescan(wif, "importANBI", false); result != nil {
				configure.Sugar.Fatal("fail to import private key")
			}
			configure.Sugar.Info("import success: ", strings.Split(splitArr[4], "=")[1])
		}
	}
	if err := scanner.Err(); err != nil {
		configure.Sugar.Fatal("scanner error: ", err.Error())
	}
}

// GetBlock get block with tx
func (btcClient *BTCRPC) GetBlock(height int64) (*btcjson.GetBlockVerboseResult, error) {
	blockHash, err := btcClient.Client.GetBlockHash(height)
	if err != nil {
		return nil, err
	}

	block, err := btcClient.Client.GetBlockVerboseTxM(blockHash)
	if err != nil {
		return nil, err
	}
	return block, nil
}

// RawSendToAddressTx btc raw tx without specify the from address
func (btcClient *BTCRPC) RawSendToAddressTx(txAmount btcutil.Amount, funbackAddress, to string, sqldb *db.GormDB) (*int64, []db.UTXO, *string, int, error) {
	var (
		utxos			[]db.UTXO
		vinAmount	int64
	)
	fee := btcutil.Amount(5000)

	// query bitcoin current best height
	binfo, err := btcClient.Client.GetBlockChainInfo()
	if err != nil {
		return nil, nil, nil, http.StatusInternalServerError, err
	}
	bheader := binfo.Headers

	confs := configure.Config.Confirmations["btc"].(int)
	sqldb.Where("height <= ? AND state = ?", bheader - int32(confs) + 1, "original").Preload("SubAddress").Find(&utxos)

	// coin select
	selectedutxos,  selectedCoins, err := CoinSelect(int64(bheader), txAmount + fee, utxos)
	if err != nil {
		return nil, nil, nil, http.StatusBadRequest, err
	}

	for _, coin := range selectedCoins.Coins() {
		vinAmount += int64(coin.Value())
	}

	feeKB, err := btcClient.Client.EstimateFee(int64(6))
	if err != nil {
		return nil, nil, nil, http.StatusBadRequest, err
	}

	funBackAddressPkScript, toPkScript, err := util.BTCWithdrawAddressValidate(funbackAddress, to)
	if err != nil {
		return nil, nil, nil, http.StatusBadRequest, err
	}

	vAmount, unSignTxHex := RawBTCTx(funBackAddressPkScript, toPkScript, feeKB, txAmount, selectedCoins)

	return &vAmount, selectedutxos, &unSignTxHex, http.StatusOK, nil
}
// RawTx btc raw tx
func (btcClient *BTCRPC) RawTx(from, to string, amountF float64, subAddress *db.SubAddress, sqldb  *db.GormDB) (*int64, []db.UTXO, *string, int, error) {
	var (
		utxos			[]db.UTXO
		vinAmount	int64
	)

	fromPkScript, toPkScript, err := util.BTCWithdrawAddressValidate(from, to)
	if err != nil {
		return nil, nil, nil, http.StatusBadRequest, err
	}

	// query bitcoin current best height
	binfo, err := btcClient.Client.GetBlockChainInfo()
	if err != nil {
		return nil, nil, nil, http.StatusInternalServerError, err
	}
	bheader := binfo.Headers

	feeKB, err := btcClient.Client.EstimateFee(int64(6))
	if err != nil {
		return nil, nil, nil, http.StatusInternalServerError, err
	}

	// query utxos, which confirmate count is more than 6
	confs := configure.Config.Confirmations["btc"].(int)
	if err = sqldb.Model(subAddress).Where("height <= ? AND state = ?", bheader - int32(confs) + 1, "original").Related(&utxos).Error; err !=nil {
		return nil, nil, nil, http.StatusNotFound, err
	}
	configure.Sugar.Info("utxos: ", utxos, " length: ", len(utxos))

	txAmount, err := btcutil.NewAmount(amountF)
	if err != nil {
		return nil, nil, nil, http.StatusBadRequest, errors.New(strings.Join([]string{"convert utxo amount(float64) to btc amount(int64 as Satoshi) error:", err.Error()}, ""))
	}
	fee := btcutil.Amount(5000)
	// coin select
	selectedutxos,  selectedCoins, err := CoinSelect(int64(bheader), txAmount + fee, utxos)
	if err != nil {
		code := http.StatusInternalServerError
		if err.Error() == "CoinSelect error: no coin selection possible" {
			code = http.StatusBadRequest
		}
		// util.GinRespException(c, code, err)
		return nil, nil, nil, code, err
	}

	for _, coin := range selectedCoins.Coins() {
		vinAmount += int64(coin.Value())
	}

	vAmount, unSignTxHex := RawBTCTx(fromPkScript, toPkScript, feeKB, txAmount, selectedCoins)
	return &vAmount, selectedutxos, &unSignTxHex, http.StatusOK, nil
}

// SendTx broadcast signed tx
func (btcClient *BTCRPC) SendTx(signedTx string, selectedUTXOs []db.UTXO, sqldb *db.GormDB) (*string, int, error) {
	tx, err := DecodeBtcTxHex(signedTx)
	if err != nil {
		e := errors.New(strings.Join([]string{"Decode signed tx error", err.Error()}, ":"))
		return nil, http.StatusInternalServerError, e
	}

	txHash, err := btcClient.Client.SendRawTransaction(tx.MsgTx(), false)
	if err != nil {
		e := errors.New(strings.Join([]string{"Bitcoin SendRawTransaction signed tx error", err.Error()}, ":"))
		return nil, http.StatusInternalServerError, e
	}
	txid := txHash.String()
	ts := sqldb.Begin()
	for _, dbutxo := range selectedUTXOs {
		ts.Model(&dbutxo).Updates(map[string]interface{}{"used_by": txid, "state": "selected"})
	}
	if err := ts.Commit().Error; err != nil {
		e := errors.New(strings.Join([]string{"update selected utxos error", err.Error()}, ":"))
		return nil, http.StatusInternalServerError, e
	}
	return &txid, http.StatusOK, nil
}

// DecodeBtcTxHex decode bitcoin transaction's hex to rawTx
func DecodeBtcTxHex(txHex string) (*btcutil.Tx, error) {
	if txHex == "" {
		hexErr := errors.New("signtx 交易签名参数错误")
		return nil, hexErr
	}
	txByte, err := hex.DecodeString(txHex)
	if err != nil {
		return nil, err
	}

	var msgTx wire.MsgTx
	if err := msgTx.Deserialize(bytes.NewReader(txByte)); err != nil {
		return nil, err
	}

	return btcutil.NewTx(&msgTx), nil
}

// SimpleCoin implements coinset Coin interface
type SimpleCoin struct {
	TxHash     *chainhash.Hash
	TxIndex    uint32
	TxValue    btcutil.Amount
	TxNumConfs int64
}

// Hash implements coinset Coin interface
func (c *SimpleCoin) Hash() *chainhash.Hash { return c.TxHash }
// Index implements coinset Coin interface
func (c *SimpleCoin) Index() uint32         { return c.TxIndex }
// Value implements coinset Coin interface
func (c *SimpleCoin) Value() btcutil.Amount { return c.TxValue }
// PkScript implements coinset Coin interface
func (c *SimpleCoin) PkScript() []byte      { return nil }
// NumConfs implements coinset Coin interface
func (c *SimpleCoin) NumConfs() int64       { return c.TxNumConfs }
// ValueAge implements coinset Coin interface
func (c *SimpleCoin) ValueAge() int64       { return int64(c.TxValue) * c.TxNumConfs }

// CoinSelect btc tx inputs
func CoinSelect(bheader int64, txAmount btcutil.Amount, utxos []db.UTXO) ([]db.UTXO, coinset.Coins, error) {
	var coins []coinset.Coin
	for _, utxo := range utxos {
		txHash, err := chainhash.NewHashFromStr(utxo.Txid)
		if err != nil {
			return nil, nil, errors.New(strings.Join([]string{"convert utxo hexTxid to txHash error: ", err.Error()}, ""))
		}
		amount, err := btcutil.NewAmount(utxo.Amount)
		if err != nil {
			return nil, nil, errors.New(strings.Join([]string{"convert utxo amount(float64) to btc amount(int64 as Satoshi) error: ", err.Error()}, ""))
		}
		coins = append(coins, coinset.Coin(&SimpleCoin{TxHash: txHash, TxIndex: utxo.VoutIndex, TxValue: amount, TxNumConfs: bheader - utxo.Height + 1}))
	}

	selector := &coinset.MaxValueAgeCoinSelector{
		MaxInputs: 50,
		MinChangeAmount: 10000,
	}

	selectedCoins, err := selector.CoinSelect(txAmount, coins)
	if err != nil {
		return nil, nil, errors.New(strings.Join([]string{"CoinSelect error: ", err.Error()}, ""))
	}
	scoins := selectedCoins.Coins()

	var selectedUTXOs []db.UTXO
	for _, coin := range scoins {
		for _, utxo := range utxos {
			if coin.Hash().String() == utxo.Txid && coin.Index() == utxo.VoutIndex {
				selectedUTXOs = append(selectedUTXOs, utxo)
			}
		}
	}
	return selectedUTXOs, selectedCoins, nil
}

// RawBTCTx btc raw tx
func RawBTCTx(funbackPkScript, toPkScript []byte, feeKB *btcjson.EstimateFeeResult, txAmount btcutil.Amount, selectedCoins coinset.Coins) (int64, string ){
	msgTx := coinset.NewMsgTxWithInputCoins(wire.TxVersion, selectedCoins)
	var vinAmount int64
	for _, coin := range selectedCoins.Coins() {
		vinAmount += int64(coin.Value())
	}

	vAmount := vinAmount

	txOutTo := wire.NewTxOut(int64(txAmount), toPkScript)
	txOutReBack := wire.NewTxOut((vinAmount-int64(txAmount)), funbackPkScript)
	msgTx.AddTxOut(txOutTo)
	msgTx.AddTxOut(txOutReBack)

	rate := mempool.SatoshiPerByte(feeKB.FeeRate)
	fee := rate.Fee(uint32(msgTx.SerializeSize()))

	if fee.String() == "0 BTC" {
		fee = btcutil.Amount(5000)
	}

	// sub tx fee
	for _, out := range msgTx.TxOut {
		if out.Value != int64(txAmount) && (vinAmount - int64(txAmount) - int64(fee)) > 0 {
			out.Value = vinAmount - int64(txAmount) - int64(fee)
		}
	}

	buf := bytes.NewBuffer(make([]byte, 0, msgTx.SerializeSize()))
	msgTx.Serialize(buf)
	return vAmount, hex.EncodeToString(buf.Bytes())
}

// GenBTCAddress generate btc address
func GenBTCAddress() (*btcutil.AddressPubKeyHash, error) {
  ldb, err := db.NewLDB("btc")
  if err != nil {
    return nil, err
  }

  seed, err := hdkeychain.GenerateSeed(hdkeychain.RecommendedSeedLen)
  if err != nil {
    return nil, errors.New(strings.Join([]string{"GenerateSeed err", err.Error()}, ":"))
  }

  key, err := hdkeychain.NewMaster(seed, &chaincfg.RegressionNetParams)
  if err != nil {
    return nil, errors.New(strings.Join([]string{"NewMaster err", err.Error()}, ":"))
  }
  add, err := key.Address(&chaincfg.RegressionNetParams)
  if err != nil {
    return nil, errors.New(strings.Join([]string{"NewAddressPubKeyHash err", err.Error()}, ":"))
  }

  _, err = ldb.Get([]byte(add.EncodeAddress()), nil)
  if err != nil && strings.Contains(err.Error(), "leveldb: not found") && key.IsPrivate(){
    priv, err := key.ECPrivKey()
    if err != nil {
      return nil, errors.New(strings.Join([]string{"master key to ec privite key error:", err.Error()}, ""))
    }

    wif, err := btcutil.NewWIF(priv, &chaincfg.RegressionNetParams, true)
    if err != nil {
      return nil, errors.New(strings.Join([]string{"btcec priv to wif:", err.Error()}, ""))
    }
    if err := ldb.Put([]byte(add.EncodeAddress()), []byte(wif.String()), nil); err != nil {
      return nil, errors.New(strings.Join([]string{"put privite key to leveldb error:", err.Error()}, ""))
    }
  }else if err != nil {
    return nil, errors.New(strings.Join([]string{"Fail to add address:", add.EncodeAddress(), " ", err.Error()}, ""))
  }
  ldb.Close()

  return add, nil
}
