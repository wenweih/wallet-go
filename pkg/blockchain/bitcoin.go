package blockchain

import (
	"fmt"
	"path/filepath"
	"bytes"
	"encoding/hex"
	"errors"
	"strings"
	"wallet-transition/pkg/configure"

	"github.com/btcsuite/btcd/btcjson"
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcd/rpcclient"
	"github.com/btcsuite/btcd/txscript"
	"github.com/btcsuite/btcd/wire"
	"github.com/btcsuite/btcutil"
	"github.com/shopspring/decimal"
	"github.com/manifoldco/promptui"
)

// BitcoinClientAlias bitcoin-core client alias
type BitcoinClientAlias struct {
	*rpcclient.Client
}

// NewbitcoinClient bitcoin rpc client
func NewbitcoinClient() *rpcclient.Client {
	connCfg := &rpcclient.ConnConfig{
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


// DumpOldWallet migrate old wallet from node
func (btcClient *BitcoinClientAlias) DumpOldWallet(serverClient *configure.ServerClient) () {
	if _, err := btcClient.DumpWallet(configure.Config.OldBTCWalletFileName); err != nil {
		if strings.Contains(err.Error(), "already exists. If you are sure this is what you want"){
			prompt := promptui.Prompt{
				Label:     strings.Join([]string{"File: ", filepath.Base(configure.Config.OldBTCWalletFileName), "backup wallet already exists, If you are sure this is what you want, move it out of the way first "}, ""),
				IsConfirm: true,
			}
			if _, err = prompt.Run(); err != nil {
				fmt.Println("pls check the old backup wallet file in", configure.Config.OldBTCWalletFileName, serverClient.SSHClient.RemoteAddr().String())
				return
			}
			if err = serverClient.SftpClient.Remove(configure.Config.OldBTCWalletFileName); err != nil {
				configure.Sugar.Fatal("Remove old backup wallet from old wallet server error: ", err.Error())
			}
			btcClient.DumpOldWallet(serverClient)
		}
	}else {
		configure.Sugar.Info("dump old btc wallet result: success")
	}
}

// GetBlock get block with tx
func (btcClient *BitcoinClientAlias) GetBlock(height int32) (*btcjson.GetBlockVerboseResult, error) {
	blockHash, err := btcClient.GetBlockHash(int64(height))
	if err != nil {
		return nil, err
	}

	block, err := btcClient.GetBlockVerboseTxM(blockHash)
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

// BtcBalanceWithID 类型
type BtcBalanceWithID struct {
	ID      string     `json:"id"`
	Balance BtcBalance `json:"balance"`
}

// BtcVoutWithID type struct
type BtcVoutWithID struct {
	ID   string
	Vout *BtcVoutES
}

// BtcVoutES type struct
type BtcVoutES struct {
	TxIDBelongTo string      `json:"txidbelongto"`
	Value        float64     `json:"value"`
	Voutindex    uint32      `json:"voutindex"`
	Coinbase     bool        `json:"coinbase"`
	Addresses    []string    `json:"addresses"`
	Height       int64       `json:"height"`
	Used         interface{} `json:"used"`
}

// BtcAddressWithValueInTx 交易中地输入输出的地址和余额
type BtcAddressWithValueInTx struct {
	Address string  `json:"address"`
	Value   float64 `json:"value"`
}

// BtcIndexUTXO vout 索引
type BtcIndexUTXO struct {
	Txid  string
	Index uint32
}

// EsBtcTx type struct
type EsBtcTx struct {
	Txid      string                    `json:"txid"`
	Fee       float64                   `json:"fee"`
	BlockHash string                    `json:"blockhash"`
	Time      int64                     `json:"time"`
	Vins      []BtcAddressWithValueInTx `json:"vins"`
	Vouts     []BtcAddressWithValueInTx `json:"vouts"`
}

// BtcvoutUsed vout used field
type BtcvoutUsed struct {
	Txid     string `json:"txid"`     // 所在交易的 id
	VinIndex uint32 `json:"vinindex"` // 作为 vin 被使用时，vin 的 vout 字段
}

// BTCBlockWithTxDetail elasticsearch 中 block Type 数据
func BTCBlockWithTxDetail(block *btcjson.GetBlockVerboseResult) interface{} {
	txs := blockTx(block.Tx)
	blockWithTx := map[string]interface{}{
		"hash":         block.Hash,
		"strippedsize": block.StrippedSize,
		"size":         block.Size,
		"weight":       block.Weight,
		"height":       block.Height,
		"versionHex":   block.VersionHex,
		"merkleroot":   block.MerkleRoot,
		"time":         block.Time,
		"nonce":        block.Nonce,
		"bits":         block.Bits,
		"difficulty":   block.Difficulty,
		"previoushash": block.PreviousHash,
		"nexthash":     block.NextHash,
		"tx":           txs,
	}
	return blockWithTx
}

func blockTx(txs []btcjson.TxRawResult) []map[string]interface{} {
	var rawTxs []map[string]interface{}
	for _, tx := range txs {
		// https://tradeblock.com/blog/bitcoin-0-8-5-released-provides-critical-bug-fixes/
		txVersion := tx.Version
		if tx.Version < 0 {
			txVersion = 1
		}
		vouts := txVouts(tx)
		vins := txVins(tx)
		rawTxs = append(rawTxs, map[string]interface{}{
			"txid":     tx.Txid,
			"hash":     tx.Hash,
			"version":  txVersion,
			"size":     tx.Size,
			"vsize":    tx.Vsize,
			"locktime": tx.LockTime,
			"vout":     vouts,
			"vin":      vins,
		})
	}
	return rawTxs
}

func txVouts(tx btcjson.TxRawResult) []map[string]interface{} {
	var vouts []map[string]interface{}
	for _, vout := range tx.Vout {
		vouts = append(vouts, map[string]interface{}{
			"value": vout.Value,
			"n":     vout.N,
			"scriptPubKey": map[string]interface{}{
				"asm":       vout.ScriptPubKey.Asm,
				"reqSigs":   vout.ScriptPubKey.ReqSigs,
				"type":      vout.ScriptPubKey.Type,
				"addresses": vout.ScriptPubKey.Addresses,
			},
		})
	}
	return vouts
}

func txVins(tx btcjson.TxRawResult) []map[string]interface{} {
	var vins []map[string]interface{}
	for _, vin := range tx.Vin {
		if len(tx.Vin) == 1 && len(vin.Coinbase) != 0 && len(vin.Txid) == 0 {
			vins = append(vins, map[string]interface{}{
				"coinbase": vin.Coinbase,
				"sequence": vin.Sequence,
			})
			break
		}
		vins = append(vins, map[string]interface{}{
			"txid": vin.Txid,
			"vout": vin.Vout,
			"scriptSig": map[string]interface{}{
				"asm": vin.ScriptSig.Asm,
			},
			"sequence": vin.Sequence,
		})
	}
	return vins
}

// get addresses in bitcoin vout
func voutAddressFun(vout btcjson.Vout) (*[]string, error) {
	var addresses []string
	if len(vout.ScriptPubKey.Addresses) > 0 {
		addresses = vout.ScriptPubKey.Addresses
		return &addresses, nil
	}
	if len(addresses) == 0 {
		return nil, errors.New("Unable to decode output address")
	}
	return nil, errors.New("address not fount in vout")
}

// NewVoutFun elasticsearch 中 voutstream Type 数据
func NewVoutFun(height int64, vout btcjson.Vout, vins []btcjson.Vin, TxID string) (*BtcVoutES, error) {
	coinbase := false
	if len(vins[0].Coinbase) != 0 && len(vins[0].Txid) == 0 {
		coinbase = true
	}
	addresses, err := voutAddressFun(vout)
	if err != nil {
		return nil, err
	}

	v := &BtcVoutES{
		TxIDBelongTo: TxID,
		Value:        vout.Value,
		Voutindex:    vout.N,
		Coinbase:     coinbase,
		Addresses:    *addresses,
		Height:       height,
		Used:         nil,
	}
	return v, nil
}

// NewBalanceJournalFun new balanceJournal instance
func NewBalanceJournalFun(address, ope, txid string, amount float64) BtcBalanceJournal {
	balancejournal := BtcBalanceJournal{
		Address: address,
		Operate: ope,
		Amount:  amount,
		Txid:    txid,
	}
	return balancejournal
}

// EsTxFun elasticsearch 中 txstream Type 数据
func EsTxFun(tx btcjson.TxRawResult, blockHash string, simpleVins, simpleVouts []BtcAddressWithValueInTx, vinAmount, voutAmount decimal.Decimal) *EsBtcTx {
	// caculate tx fee
	fee := vinAmount.Sub(voutAmount)
	if len(tx.Vin) == 1 && len(tx.Vin[0].Coinbase) != 0 && len(tx.Vin[0].Txid) == 0 || vinAmount.Equal(voutAmount) {
		fee = decimal.NewFromFloat(0)
	}
	// bulk insert tx docutment
	esFee, _ := fee.Float64()
	result := &EsBtcTx{
		Txid:      tx.Txid,
		Fee:       esFee,
		BlockHash: blockHash,
		Time:      tx.Time, // TODO: time field is nil, need to fix
		Vins:      simpleVins,
		Vouts:     simpleVouts,
	}
	return result
}

// ParseTxVout parse bitcoin vout and construct data
func ParseTxVout(vout btcjson.Vout, txid string) ([]BtcAddressWithValueInTx, []interface{}, []BtcBalance, []BtcAddressWithAmountAndTxid) {
	var (
		txVoutsField                      []BtcAddressWithValueInTx
		voutAddresses                     []interface{} // All addresses related with vout in a block
		voutAddressWithAmounts            []BtcBalance
		voutAddressWithAmountAndTxidSlice []BtcAddressWithAmountAndTxid
	)
	// vouts field in tx type
	for _, address := range vout.ScriptPubKey.Addresses {
		txVoutsField = append(txVoutsField, BtcAddressWithValueInTx{
			Address: address,
			Value:   vout.Value,
		})

		// vout addresses slice
		voutAddresses = append(voutAddresses, address)
		// vout addresses with amount
		voutAddressWithAmounts = append(voutAddressWithAmounts, BtcBalance{address, vout.Value})
		voutAddressWithAmountAndTxidSlice = append(voutAddressWithAmountAndTxidSlice, BtcAddressWithAmountAndTxid{
			Address: address, Amount: vout.Value, Txid: txid})
	}
	return txVoutsField, voutAddresses, voutAddressWithAmounts, voutAddressWithAmountAndTxidSlice
}

// ParseESVout parse es vout and construct data
func ParseESVout(voutWithID BtcVoutWithID, txid string) ([]BtcAddressWithValueInTx, []interface{}, []BtcBalance, []BtcAddressWithAmountAndTxid) {
	var (
		txTypeVinsField                  []BtcAddressWithValueInTx
		vinAddresses                     []interface{}
		vinAddressWithAmountSlice        []BtcBalance
		vinAddressWithAmountAndTxidSlice []BtcAddressWithAmountAndTxid
	)

	for _, address := range voutWithID.Vout.Addresses {
		vinAddresses = append(vinAddresses, address)
		vinAddressWithAmountSlice = append(vinAddressWithAmountSlice, BtcBalance{address, voutWithID.Vout.Value})
		txTypeVinsField = append(txTypeVinsField, BtcAddressWithValueInTx{address, voutWithID.Vout.Value})
		vinAddressWithAmountAndTxidSlice = append(vinAddressWithAmountAndTxidSlice, BtcAddressWithAmountAndTxid{
			Address: address, Amount: voutWithID.Vout.Value, Txid: txid})
	}
	return txTypeVinsField, vinAddresses, vinAddressWithAmountSlice, vinAddressWithAmountAndTxidSlice
}

// IndexedVinsFun index vins
func IndexedVinsFun(vins []btcjson.Vin) []BtcIndexUTXO {
	var IndexUTXOs []BtcIndexUTXO
	for _, vin := range vins {
		item := BtcIndexUTXO{vin.Txid, vin.Vout}
		IndexUTXOs = append(IndexUTXOs, item)
	}
	return IndexUTXOs
}

// IndexedVoutsFun index vouts
func IndexedVoutsFun(vouts []btcjson.Vout, txid string) []BtcIndexUTXO {
	var IndexUTXOs []BtcIndexUTXO
	for _, vout := range vouts {
		IndexUTXOs = append(IndexUTXOs, BtcIndexUTXO{txid, vout.N})
	}
	return IndexUTXOs
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
