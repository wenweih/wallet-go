package main

import (
  "errors"
  "reflect"
  "strconv"
  "strings"
  "net/http"
  "math/big"
  "github.com/gin-gonic/gin"
  "wallet-transition/pkg/util"
  "wallet-transition/pkg/configure"
  "github.com/ethereum/go-ethereum/common"
  "github.com/ethereum/go-ethereum/core/types"
	"github.com/btcsuite/btcutil"
	"github.com/btcsuite/btcd/chaincfg"
  "github.com/btcsuite/btcd/chaincfg/chainhash"
)

func txHandle(c *gin.Context)  {
  asset, _ := c.Get("asset")
  detailParams, _ := c.Get("detail")
  params := reflect.ValueOf(detailParams)
  txParams := util.TxParams{}
  if params.Kind() == reflect.Map {
    for _, key := range params.MapKeys() {
      switch key.Interface() {
      case "txid":
        txParams.Txid = params.MapIndex(key).Interface().(string)
      }
    }
  }else {
    util.GinRespException(c, http.StatusBadRequest, errors.New("detail params error"))
    return
  }

  switch asset.(string) {
  case "eth":
    tx, _, err := ethClient.Client.TransactionByHash(c, common.HexToHash(txParams.Txid))
    if err != nil {
      util.GinRespException(c, http.StatusBadRequest, err)
      return
    }
    c.JSON(http.StatusOK, gin.H {
      "status": http.StatusOK,
      "tx": tx,
    })
    return
  case "btc":
    txHash, err := chainhash.NewHashFromStr(txParams.Txid)
    if err != nil {
      util.GinRespException(c, http.StatusBadRequest, err)
      return
    }
    tx, err := btcClient.Client.GetRawTransaction(txHash)
    if err != nil {
      util.GinRespException(c, http.StatusBadRequest, err)
      return
    }
    configure.Sugar.Info("xffff: ", tx.MsgTx())
    c.JSON(http.StatusOK, gin.H {
      "status": http.StatusOK,
      "tx": tx.MsgTx(),
    })
    return
  }
}

func blockHandle(c *gin.Context)  {
  asset, _ := c.Get("asset")
  detailParams, _ := c.Get("detail")
  params := reflect.ValueOf(detailParams)
  blockParams := util.BlockParams{}
  if params.Kind() == reflect.Map {
    for _, key := range params.MapKeys() {
      switch key.Interface() {
      case "height":
        blockParams.Height = params.MapIndex(key).Interface().(string)
      }
    }
  }else {
    util.GinRespException(c, http.StatusBadRequest, errors.New("detail params error"))
    return
  }
  switch asset.(string) {
  case "btc":
    height, err := strconv.ParseInt(blockParams.Height, 10, 64)
    if err != nil {
      util.GinRespException(c, http.StatusBadRequest, errors.New("height param error"))
      return
    }
    block, err := btcClient.GetBlock(height)
    if err !=nil {
      util.GinRespException(c, http.StatusInternalServerError, err)
      return
    }
    c.JSON(http.StatusOK, gin.H {
      "status": http.StatusOK,
      "block": block,
    })
    return
  case "eth":
    height, ok := new(big.Int).SetString(blockParams.Height, 10)
    if !ok {
      util.GinRespException(c, http.StatusBadRequest, errors.New("height param error"))
      return
    }
    ethBlock, err := ethClient.Client.BlockByNumber(c, height)
    if err !=nil {
      util.GinRespException(c, http.StatusInternalServerError, err)
      return
    }
    var block struct{Header types.Header; Tx []*types.Transaction}
    block.Header = *ethBlock.Header()
    block.Tx = ethBlock.Body().Transactions
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

  if !common.IsHexAddress(balanceParams.Address) {
    err := errors.New(strings.Join([]string{"To: ", balanceParams.Address, " isn't valid ethereum address"}, ""))
    util.GinRespException(c, http.StatusBadRequest, err)
    return
  }

  if asset.(string) == "eth" {
    balance, err := ethClient.Client.BalanceAt(c, common.HexToAddress(balanceParams.Address), nil)
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
    _, err := btcutil.DecodeAddress(*addressHex, &chaincfg.TestNet3Params)
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

func bestBlock(c *gin.Context)  {
  asset, _ := c.Get("asset")
  switch asset.(string) {
  case "eth":
    block, err := ethClient.Client.BlockByNumber(c, nil)
    if err != nil {
      util.GinRespException(c, http.StatusBadRequest, err)
      return
    }
    c.JSON(http.StatusOK, gin.H {
      "status": http.StatusOK,
      "block_number": block.NumberU64(),
    })
    return
  case "btc":
    btcInfo, err := btcClient.Client.GetBlockChainInfo()
    if err != nil {
      util.GinRespException(c, http.StatusBadRequest, err)
      return
    }
    c.JSON(http.StatusOK, gin.H {
      "status": http.StatusOK,
      "block_number": btcInfo.Headers,
    })
  default:
    util.GinRespException(c, http.StatusBadRequest, errors.New("Only support ethereum and bitcoin"))
    return
  }
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
