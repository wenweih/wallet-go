package blockchain

import (
	"os"
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"strings"
	"path/filepath"
	"encoding/hex"
	"wallet-transition/pkg/db"
	"wallet-transition/pkg/util"
	"wallet-transition/pkg/configure"
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/btcjson"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcd/rpcclient"
	"github.com/btcsuite/btcd/txscript"
	"github.com/btcsuite/btcd/wire"
	"github.com/btcsuite/btcutil"
	"github.com/manifoldco/promptui"
	"github.com/shopspring/decimal"
	"github.com/btcsuite/btcd/mempool"
	"github.com/btcsuite/btcutil/coinset"
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

// BtcUTXO utxo type
type BtcUTXO struct {
	Txid      string  `json:"txid"`
	Amount    float64 `json:"amount"`
	Height    int64   `json:"height"`
	VoutIndex uint32  `json:"voutindex"`
}

// DumpBTC dump wallet from node
func (btcClient *BTCRPC) DumpBTC(local bool) {
	oldWalletServerClient, err := util.NewServerClient(configure.Config.OldBTCWalletServerUser, configure.Config.OldBTCWalletServerPass, configure.Config.OldBTCWalletServerHost)
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
				Label:     strings.Join([]string{"File: ", filepath.Base(configure.Config.BackupWalletPath), "backup wallet already exists, If you are sure this is what you want, move it out of the way first "}, ""),
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

// BtcBalance type struct
type BtcBalance struct {
	Address string  `json:"address"`
	Amount  float64 `json:"amount"`
}

// BtcBalanceJournal 余额变更流水
type BtcBalanceJournal struct {
	Address string  `json:"address"`
	Amount  float64 `json:"amount"`
	Operate string  `json:"operate"`
	Txid    string  `json:"txid"`
}

// BtcAddressWithAmount 地址-余额类型
type BtcAddressWithAmount struct {
	Address string          `json:"address"`
	Amount  decimal.Decimal `json:"amount"`
}

// BtcAddressWithAmountAndTxid 地址-余额类型
type BtcAddressWithAmountAndTxid struct {
	Address string  `json:"address"`
	Amount  float64 `json:"amount"`
	Txid    string  `json:"txid"`
}

// CreateRawBTCTx create raw tx: vins related with one address, vouts related to one address
func CreateRawBTCTx(from, to string, value, fee float64, utxos []*BtcUTXO) (*string, error) {
	fromAddress, err := btcutil.DecodeAddress(from, &chaincfg.RegressionNetParams)
	if err != nil {
		return nil, errors.New("DecodeAddress from address error")
	}
	fromPkScript, err := txscript.PayToAddrScript(fromAddress)
	if err != nil {
		return nil, errors.New("from address PayToAddrScript error")
	}

	toAddress, err := btcutil.DecodeAddress(to, &chaincfg.RegressionNetParams)
	if err != nil {
		return nil, errors.New("DecodeAddress to address error")
	}
	toPkScript, err := txscript.PayToAddrScript(toAddress)
	if err != nil {
		return nil, errors.New("to address PayToAddrScript error")
	}

	var vinAmount float64
	tx := wire.NewMsgTx(wire.TxVersion)
	for _, utxo := range utxos {
		txHash, err := chainhash.NewHashFromStr(utxo.Txid)
		if err != nil {
			return nil, errors.New(strings.Join([]string{"Txid to hash error", err.Error()}, ":"))
		}
		outPoint := wire.NewOutPoint(txHash, utxo.VoutIndex)
		txIn := wire.NewTxIn(outPoint, nil, nil)
		tx.AddTxIn(txIn)
		vinAmount += utxo.Amount
	}

	if vinAmount < value+fee {
		return nil, errors.New("value + fee is more than vins' amount")
	}
	txOutTo := wire.NewTxOut(int64(value*100000000), toPkScript)
	txOutReBack := wire.NewTxOut(int64((vinAmount-value-fee)*100000000), fromPkScript)
	tx.AddTxOut(txOutTo)
	tx.AddTxOut(txOutReBack)
	// txToHex
	buf := bytes.NewBuffer(make([]byte, 0, tx.SerializeSize()))
	tx.Serialize(buf)
	txHex := hex.EncodeToString(buf.Bytes())
	return &txHex, nil
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
			return nil, nil, errors.New(strings.Join([]string{"onvert utxo amount(float64) to btc amount(int64 as Satoshi) error: ", err.Error()}, ""))
		}
		coins = append(coins, coinset.Coin(&SimpleCoin{TxHash: txHash, TxIndex: utxo.VoutIndex, TxValue: amount, TxNumConfs: bheader - utxo.Height + 1}))
	}

	selector := &coinset.MaxValueAgeCoinSelector{
		MaxInputs: 10,
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
func RawBTCTx(fromPkScript, toPkScript []byte, feeKB *btcjson.EstimateFeeResult, txAmount btcutil.Amount, selectedCoins coinset.Coins) string {
	msgTx := coinset.NewMsgTxWithInputCoins(wire.TxVersion, selectedCoins)
	var vinAmount int64
	for _, coin := range selectedCoins.Coins() {
		vinAmount += int64(coin.Value())
	}

	txOutTo := wire.NewTxOut(int64(txAmount), toPkScript)
	txOutReBack := wire.NewTxOut((vinAmount-int64(txAmount)), fromPkScript)
	msgTx.AddTxOut(txOutTo)
	msgTx.AddTxOut(txOutReBack)

	rate := mempool.SatoshiPerByte(feeKB.FeeRate)
	fee := rate.Fee(uint32(msgTx.SerializeSize()))

	// sub tx fee
	for _, out := range msgTx.TxOut {
		if out.Value != int64(txAmount) && (vinAmount - int64(txAmount) - int64(fee)) > 0{
			out.Value = vinAmount - int64(txAmount) - int64(fee)
		}
	}

	buf := bytes.NewBuffer(make([]byte, 0, msgTx.SerializeSize()))
	msgTx.Serialize(buf)
	return hex.EncodeToString(buf.Bytes())
}
