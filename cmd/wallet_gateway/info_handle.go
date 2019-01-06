package main

import (
  "context"
  "errors"
  "reflect"
  "strings"
  "net/http"
  "github.com/gin-gonic/gin"
  "wallet-transition/pkg/util"
  "wallet-transition/pkg/configure"
  "github.com/ethereum/go-ethereum/common"
	"github.com/btcsuite/btcutil"
	"github.com/btcsuite/btcd/chaincfg"
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

  keys := make([]string, len(configure.Config.ETHToken))
  keys = append(keys, "eth")
  for k := range configure.Config.ETHToken {
    keys = append(keys, k)
  }
  if !util.Contain(asset.(string), keys) {
    util.GinRespException(c, http.StatusBadRequest, errors.New(strings.Join([]string{asset.(string), " balance query is not be supported"}, "")))
    return
  }

  if balanceParams.Address == "" {
    util.GinRespException(c, http.StatusBadRequest, errors.New("address param is required"))
    return
  }

  if asset.(string) == "eth" {
    balance, err := ethClient.Client.BalanceAt(context.Background(), common.HexToAddress(balanceParams.Address), nil)
  	if err != nil {
      util.GinRespException(c, http.StatusInternalServerError, err)
      return
  	}
    c.JSON(http.StatusOK, gin.H {
      "status": http.StatusOK,
      "balance": util.ToEther(balance).String(),
    })
    return
  }

  balance, err := ethClient.GetTokenBalance(asset.(string), balanceParams.Address)
  if err != nil {
    util.GinRespException(c, http.StatusInternalServerError, err)
    return
  }
  c.JSON(http.StatusOK, gin.H {
    "status": http.StatusOK,
    "balance": util.ToEther(balance).String(),
  })
}

func addressValidator(c *gin.Context) {
  asset, _ := c.Get("asset")
  detailParams, _ := c.Get("detail")
  addressHex, err := addressWithAssetParams(detailParams)
  if err != nil {
    util.GinRespException(c, http.StatusBadRequest, err)
    return
  }
  switch asset.(string) {
  case "btc":
    _, err := btcutil.DecodeAddress(*addressHex, &chaincfg.RegressionNetParams)
    if err != nil {
      e := errors.New(strings.Join([]string{"To address illegal", err.Error()}, ":"))
      util.GinRespException(c, http.StatusBadRequest, e)
      return
    }
  case "eth":
    if !common.IsHexAddress(*addressHex) {
      err := errors.New(strings.Join([]string{"To: ", *addressHex, " isn't valid ethereum address"}, ""))
      util.GinRespException(c, http.StatusBadRequest, err)
      return
    }
  default:
    util.GinRespException(c, http.StatusBadRequest, errors.New("Only support ethereum and bitcoin address validate"))
    return
  }

  c.JSON(http.StatusOK, gin.H {
    "status": http.StatusOK,
    "valid": true,
  })
}

func addressWithAssetParams(paramsI interface{}) (*string, error) {
  params := reflect.ValueOf(paramsI)
  addressAsset := util.AssetWithAddress{}
  if params.Kind() == reflect.Map {
    for _, key := range params.MapKeys() {
      switch key.Interface() {
      case "address":
        addressAsset.Address = params.MapIndex(key).Interface().(string)
      }
    }
  }else {
    return nil, errors.New("detail params error")
  }

  if addressAsset.Address == "" {
    return nil, errors.New("address param is required")
  }
  return &addressAsset.Address, nil
}
