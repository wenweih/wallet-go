package main

import (
  "errors"
  "strconv"
  "strings"
  "net/http"
  "math/big"
  "encoding/json"
  "github.com/gin-gonic/gin"
  "wallet-transition/pkg/util"
  "wallet-transition/pkg/configure"
  "github.com/ethereum/go-ethereum/common"
  "github.com/ethereum/go-ethereum/core/types"
	"github.com/btcsuite/btcutil"
  "github.com/btcsuite/btcd/chaincfg/chainhash"
)

func txHandle(c *gin.Context)  {
  asset, _ := c.Get("asset")
  detailParams, _ := c.Get("detail")

  var txParams util.TxParams
  if err := json.Unmarshal(detailParams.([]byte), &txParams); err != nil {
    util.GinRespException(c, http.StatusBadRequest, err)
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

  var blockParams util.BlockParams
  if err := json.Unmarshal(detailParams.([]byte), &blockParams); err != nil {
    util.GinRespException(c, http.StatusBadRequest, err)
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

func omniBalanceHandle(c *gin.Context)  {
  asset, _ := c.Get("asset")
  detailParams, _ := c.Get("detail")

  balanceParams, err := balanceParamsH("omnicore", asset.(string), detailParams.([]byte))
  if err != nil {
    util.GinRespException(c, http.StatusBadRequest, err)
    return
  }

  propertyid := configure.Config.OmniToken[balanceParams.Asset].(int)

  bal, err := omniClient.GetOmniBalance(balanceParams.Address, propertyid)
  if err != nil {
    util.GinRespException(c, http.StatusInternalServerError, err)
    return
  }
  c.JSON(http.StatusOK, gin.H {
    "status": http.StatusOK,
    "balance": bal.Balance,
  })
}

func addressValidator(c *gin.Context) {
  asset, _ := c.Get("asset")
  detailParams, _ := c.Get("detail")
  addressHex, err := addressWithAssetParams(detailParams.([]byte))
  if err != nil {
    util.GinRespException(c, http.StatusBadRequest, err)
    return
  }
  switch asset.(string) {
  case "btc":
    _, err := btcutil.DecodeAddress(*addressHex, bitcoinnet)
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

func addressWithAssetParams(params []byte) (*string, error) {
  var addressAsset util.AssetWithAddress
  if err := json.Unmarshal(params, &addressAsset); err != nil {
    return nil, err
  }

  if addressAsset.Address == "" {
    return nil, errors.New("address param is required")
  }
  return &addressAsset.Address, nil
}
