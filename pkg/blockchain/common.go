package blockchain

import (
  "context"
  "errors"
  "net/http"
  "wallet-transition/pkg/db"
  "github.com/btcsuite/btcd/chaincfg"
)

// RawTx raw transaction for withdraw endpoint
func RawTx(ctx context.Context, from, to, asset string, amount float64, subAddress *db.SubAddress, btcClient *BTCRPC, ethClient *ETHRPC, sqldb *db.GormDB, bitcoinnet *chaincfg.Params) (*string, *string, *int64, []db.UTXO, int, error) {
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
  case "eth":
    netVersion, rawTxHex, err := ethClient.RawTx(ctx, from, to, amount)
    if err != nil {
      return nil, nil, nil, nil, http.StatusBadRequest, err
    }
    chainID = *netVersion
    unSignTxHex = *rawTxHex
  case "abb", "abb2", "sb":
    netVersion, rawTxHex, err := ethClient.RawTokenTx(ctx, from, to, asset, amount)
    if err != nil {
      return nil, nil, nil, nil, http.StatusBadRequest, err
    }
    chainID = *netVersion
    unSignTxHex = *rawTxHex
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
func SendTx(ctx context.Context, asset, hexSignedTx string, selectedUTXOs []db.UTXO, btcClient *BTCRPC, ethClient *ETHRPC, sqldb   *db.GormDB) (*string, int, error) {
  txid := ""
  switch asset {
  case "btc":
    btcTxid, httpStatus, err := btcClient.SendTx(hexSignedTx, selectedUTXOs, sqldb)
    if err != nil {
      return nil, httpStatus, err
    }
    txid = *btcTxid
  case "eth":
    ethTxid, err := ethClient.SendTx(ctx, hexSignedTx)
    if err != nil {
      return nil, http.StatusInternalServerError, err
    }
    txid = *ethTxid
  default:
    return nil, http.StatusBadRequest, errors.New("Unsupported asset")
  }
  return &txid, http.StatusOK, nil
}
