package blockchain

import (
  "context"
  "errors"
  "net/http"
  "wallet-transition/pkg/db"
  "github.com/btcsuite/btcd/chaincfg"
)

// NewBlockchain chain object
func NewBlockchain(wallet ChainWallet, operator TxOperator, query ChainQuery) *Blockchain {
  return &Blockchain{Wallet: wallet, Operator: operator, Query: query}
}

// RawTx raw transaction for withdraw endpoint
func RawTx(ctx context.Context, from, to, asset string, amount float64, subAddress *db.SubAddress, btcClient *BTCRPC, sqldb *db.GormDB, bitcoinnet *chaincfg.Params) (*string, *string, *int64, []db.UTXO, int, error) {
  var (
    chainID     string
    vinAmount   int64
    unSignTxHex string
    selectedUTXOs []db.UTXO
  )
  // raw tx
  switch asset {
  case "btc":
    vAmount, selectedutxos, rawTxHex, httpStatus, err := btcClient.RawTx(from, to, amount, subAddress, sqldb, false, bitcoinnet)
    if err != nil {
      return nil, nil, nil, nil, httpStatus, err
    }
    selectedUTXOs = selectedutxos
    unSignTxHex = *rawTxHex
    vinAmount = *vAmount
  case "omni_first_token":
    vAmount, selectedutxos, rawTxHex, httpStatus, err := btcClient.RawTx(from, to, amount, subAddress, sqldb, true, bitcoinnet)
    if err != nil {
      return nil, nil, nil, nil, httpStatus, err
    }
    selectedUTXOs = selectedutxos
    unSignTxHex = *rawTxHex
    vinAmount = *vAmount
  }
  return &unSignTxHex, &chainID, &vinAmount, selectedUTXOs, http.StatusOK, nil
}

// SendTx broadcast tx
func SendTx(ctx context.Context, asset, hexSignedTx string, selectedUTXOs []db.UTXO, btcClient *BTCRPC, sqldb   *db.GormDB) (*string, int, error) {
  txid := ""
  switch asset {
  case "btc":
    btcTxid, httpStatus, err := btcClient.SendTx(hexSignedTx, selectedUTXOs, sqldb)
    if err != nil {
      return nil, httpStatus, err
    }
    txid = *btcTxid
  default:
    return nil, http.StatusBadRequest, errors.New("Unsupported asset")
  }
  return &txid, http.StatusOK, nil
}
