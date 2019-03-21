package main

import (
  "fmt"
  "net/http"
  "encoding/json"
  "wallet-go/pkg/util"
  "github.com/gin-gonic/gin"
  "wallet-go/pkg/configure"
  "github.com/btcsuite/btcd/chaincfg/chainhash"
)

func btcBestBlockNotifyHandle(c *gin.Context) {
  strHash := c.Query("hash")
  blockHash, err := chainhash.NewHashFromStr(strHash)
  if err != nil {
    configure.Sugar.DPanic(err.Error())
    util.GinRespException(c, http.StatusInternalServerError, fmt.Errorf("NewHashFromStr %s", err))
    return
  }
  rawBlock, err := btcClient.Client.GetBlockVerboseTxM(blockHash)
  if err != nil {
    configure.Sugar.DPanic(err.Error())
    util.GinRespException(c, http.StatusInternalServerError, fmt.Errorf("GetBlockVerboseTxM %s", err))
    return
  }
  body, err := json.Marshal(rawBlock)
  if err != nil {
    configure.Sugar.Warn("json Marshal raw block error", err.Error())
  }
  messageClient.Publish(body, "bestblock", "fanout", "bitcoincore", "bitcoincore_best_block_queue")
  c.JSON(http.StatusOK, gin.H {
    "status": http.StatusOK,
    "address": "hi",
  })
}
