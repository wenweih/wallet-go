package blockchain

import (
  // "errors"
  "net/http"
  "wallet-transition/pkg/db"
)

// RawTx raw transaction for withdraw endpoint
func RawTx(from, to, asset string, amount float64, subAddress *db.SubAddress, btcClient *BTCRPC, ethClient *ETHRPC, sqldb   *db.GormDB) (*string, *string, *int64, []db.UTXO, int, error) {
  var (
    chainID     string
    vinAmount   int64
    unSignTxHex string
    selectedUTXOs []db.UTXO
  )
  // raw tx
  switch asset {
  case "btc":
    vAmount, selectedutxos, rawTxHex, httpStatus, err := btcClient.RawTx(from, to, amount, subAddress, sqldb)
    if err != nil {
      return nil, nil, nil, nil, httpStatus, err
    }
    selectedUTXOs = selectedutxos
    unSignTxHex = *rawTxHex
    vinAmount = *vAmount
  case "eth":
    netVersion, rawTxHex, err := ethClient.RawTx(from, to, amount)
    if err != nil {
      return nil, nil, nil, nil, http.StatusBadRequest, err
    }
    chainID = *netVersion
    unSignTxHex = *rawTxHex
  }
  return &unSignTxHex, &chainID, &vinAmount, selectedUTXOs, http.StatusOK, nil
}
