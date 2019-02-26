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
  pb "wallet-transition/pkg/pb"
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
  accountMap := configure.ChainsInfo[blockchain.EOSIO].Accounts
  var fromName string
  for name := range accountMap {
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


  eosioInfo, err := eosChain.Client.GetInfo()
  if err != nil {
    util.GinRespException(c, http.StatusBadRequest, err)
    return
  }

  pubkey := accountMap[fromName]
  res, err := grpcClient.SignatureEOSIO(c, &pb.SignatureEOSIOReq{Pubkey: pubkey, RawTxHex: rawTxHex, ChainID: eosioInfo.ChainID.String()})
  if err != nil {
    util.GinRespException(c, http.StatusBadRequest, err)
    return
  }

  txid, err := b.Operator.BroadcastTx(res.HexSignedTx)
  if err != nil {
    util.GinRespException(c, http.StatusInternalServerError, err)
    return
  }

  configure.Sugar.Info("txid: ", txid)

}
