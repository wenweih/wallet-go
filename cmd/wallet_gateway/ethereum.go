package main

import (
  "net/http"
  "github.com/gin-gonic/gin"
  "wallet-transition/pkg/util"
)

func ethereumBalanceHandle(c *gin.Context)  {
  asset, _ := c.Get("asset")
  detailParams, _ := c.Get("detail")

  balanceParams, err := balanceParamsH("ethereum", asset.(string), detailParams.([]byte))
  if err != nil {
    util.GinRespException(c, http.StatusBadRequest, err)
    return
  }

  balance, err := ethClient.GetEthereumBalance(balanceParams.Asset, balanceParams.Address)
  if err != nil {
    util.GinRespException(c, http.StatusInternalServerError, err)
    return
  }
  c.JSON(http.StatusOK, gin.H {
    "status": http.StatusOK,
    "balance": util.ToEther(balance).String(),
  })
}
