package main

import (
  "fmt"
  "errors"
  "strings"
  "net/http"
  "encoding/json"
  "github.com/gin-gonic/gin"
  "wallet-transition/pkg/util"
  "wallet-transition/pkg/configure"
  "wallet-transition/pkg/blockchain"
  "github.com/eoscanada/eos-go"
)

func eosioBalanceHandle(c *gin.Context) {
  eosChain := blockchain.EOSChain{Client: eosClient}
  b := blockchain.NewBlockchain(nil, nil, eosChain)
  bal, err := b.Query.Balance("huangwenwei", "EOS", "eosio.token")
  if err !=nil {
    util.GinRespException(c, http.StatusBadRequest, err)
    return
  }
  c.JSON(http.StatusOK, gin.H {
    "status": http.StatusOK,
    "balance": bal,
  })
}

func eosiotxHandle(c *gin.Context) {
  assetParams, _ := c.Get("asset")
  detailParams, _ := c.Get("detail")

  eosChain := blockchain.EOSChain{Client: eosClient}
  b := blockchain.NewBlockchain(nil, eosChain, eosChain)

  if configure.ChainAssets[assetParams.(string)] != blockchain.EOSIO {
    util.GinRespException(c, http.StatusBadRequest, errors.New("asset params error, should be eos or eos token asset"))
    return
  }

  var params util.EOSIOtxParams
  if err := json.Unmarshal(detailParams.([]byte), &params); err != nil {
    util.GinRespException(c, http.StatusBadRequest, err)
    return
  }

  paramsQuantity, err := eos.NewAsset(params.Amount)
  if err != nil {
    util.GinRespException(c, http.StatusInternalServerError, fmt.Errorf("paramsQuantity error: %s", err))
    return
  }

  contract := configure.ChainsInfo[blockchain.EOSIO].Tokens[strings.ToLower(params.Asset)]
  ammountMap := configure.ChainsInfo[blockchain.EOSIO].Accounts
  var fromName string
  for name := range ammountMap {
    bal, err := b.Query.Balance(name, params.Asset, contract)
    if err != nil {
      util.GinRespException(c, http.StatusInternalServerError, fmt.Errorf("get account balance error: %s", err))
      return
    }
    balQuantity, err := eos.NewAsset(bal)
    if err != nil {
      util.GinRespException(c, http.StatusInternalServerError, fmt.Errorf("eos quantity error: %s", err))
      return
    }
    if paramsQuantity.Amount > balQuantity.Amount {
      util.GinRespException(c, http.StatusBadRequest, fmt.Errorf("quanyify error: %d : %d", paramsQuantity.Amount, balQuantity.Amount))
      return
    }
    fromName = name
  }
  rawTxHex, err := b.Operator.RawTx(fromName, params.Receiptor, params.Amount, params.Memo, params.Asset)
  if err != nil {
    util.GinRespException(c, http.StatusBadRequest, err)
    return
  }

  configure.Sugar.Info("rawTxHex: ", rawTxHex)
}
