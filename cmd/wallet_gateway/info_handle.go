package main

import (
  "errors"
  "reflect"
  "net/http"
  "github.com/gin-gonic/gin"
  "wallet-transition/pkg/util"
)

func blockHandle(c *gin.Context)  {
  asset, _ := c.Get("asset")
  detailParams, _ := c.Get("detail")
  params := reflect.ValueOf(detailParams)
  blockParams := util.BlockParams{}
  if params.Kind() == reflect.Map {
    for _, key := range params.MapKeys() {
      switch key.Interface() {
      case "height":
        blockParams.Height = params.MapIndex(key).Interface().(int64)
      }
    }
  }else {
    util.GinRespException(c, http.StatusBadRequest, errors.New("detail params error"))
    return
  }
  switch asset.(string) {
  case "btc":
    block, err := btcClient.GetBlock(blockParams.Height)
    if err !=nil {
      util.GinRespException(c, http.StatusInternalServerError, err)
      return
    }
    c.JSON(http.StatusOK, gin.H {
      "status": http.StatusOK,
      "block": block,
    })
    return
  }
}

func balanceHandle(c *gin.Context)  {
  asset, _ := c.Get("asset")
  detailParams, _ := c.Get("detail")
  params := reflect.ValueOf(detailParams)
  balanceParams := util.BalanceParams{}
  if params.Kind() == reflect.Map {
    for _, key := range params.MapKeys() {
      switch key.Interface() {
      case "address":
        balanceParams.Address = params.MapIndex(key).Interface().(string)
      }
    }
  }else {
    util.GinRespException(c, http.StatusBadRequest, errors.New("detail params error"))
    return
  }

  balance, err := ethClient.GetTokenBalance(asset.(string), balanceParams.Address)
  if err != nil {
    util.GinRespException(c, http.StatusInternalServerError, err)
    return
  }
  c.JSON(http.StatusOK, gin.H {
    "status": http.StatusOK,
    "balance": balance.String(),
  })

}
