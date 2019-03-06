package main

import (
  "fmt"
  "net/http"
  "encoding/json"
  "github.com/gin-gonic/gin"
  "wallet-transition/pkg/configure"
  "wallet-transition/pkg/util"
  "wallet-transition/pkg/blockchain"
  "github.com/btcsuite/btcutil"
)

func omniBalanceHandle(c *gin.Context) {
  asset, _ := c.Get("asset")
  detailParams, _ := c.Get("detail")

  if configure.ChainAssets[asset.(string)] != blockchain.Bitcoin {
    util.GinRespException(c, http.StatusBadRequest, fmt.Errorf("%s is't Bitcoin asset", asset.(string)))
    return
  }

  var balanceParams util.BalanceParams
  if err := json.Unmarshal(detailParams.([]byte), &balanceParams); err != nil {
    util.GinRespException(c, http.StatusInternalServerError, err)
    return
  }

  if balanceParams.Address == "" {
    util.GinRespException(c, http.StatusInternalServerError, fmt.Errorf("address param is required"))
    return
  }

  _, err := btcutil.DecodeAddress(balanceParams.Address, bitcoinnet)
  if err != nil {
    util.GinRespException(c, http.StatusBadRequest, fmt.Errorf("%s is illegal Bitcoin Address, %s", asset.(string), err))
    return
  }

  chain := blockchain.BitcoinCoreChain{Mode: bitcoinnet, Client: omniClient}
  b := blockchain.NewBlockchain(nil, nil, chain)
  bal, err := b.Query.Balance(c, balanceParams.Address, balanceParams.Asset, "")
  if err != nil {
    util.GinRespException(c, http.StatusInternalServerError, err)
    return
  }

  c.JSON(http.StatusOK, gin.H {
    "status": http.StatusOK,
    "balance": bal,
  })
}
