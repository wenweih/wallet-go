package main

import (
  "fmt"
  "strings"
  "net/http"
  "encoding/json"
  "github.com/gin-gonic/gin"
  "wallet-go/pkg/db"
  "wallet-go/pkg/util"
  "wallet-go/pkg/configure"
  "wallet-go/pkg/blockchain"
  "github.com/eoscanada/eos-go"
  pb "wallet-go/pkg/pb"
  empty "github.com/golang/protobuf/ptypes/empty"
)

func eosioWalletHandle(c *gin.Context) {
  asset, _ := c.Get("asset")
  chain := configure.ChainAssets[asset.(string)]
  if chain == blockchain.EOSIO {
    res, err := grpcClient.EOSIOWallet(c, &empty.Empty{})
    if err != nil {
      util.GinRespException(c, http.StatusInternalServerError, err)
      return
    }
    address := res.Address
    if err := sqldb.Create(&db.SubAddress{Address: address, Asset: blockchain.EOSIO}).Error; err != nil {
      util.GinRespException(c, http.StatusInternalServerError, err)
      return
    }
    c.JSON(http.StatusOK, gin.H {
      "status": http.StatusOK,
      "address": address,
    })
  }else {
    util.GinRespException(c, http.StatusBadRequest, fmt.Errorf("%s is't EOSIO token", asset.(string)))
    return
  }
}

func eosioBalanceHandle(c *gin.Context) {
  eosChain := blockchain.EOSChain{Client: eosClient}
  b := blockchain.NewBlockchain(nil, nil, eosChain)
  bal, err := b.Query.Balance(c, "huangwenwei", "EOS", "eosio.token")
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
    util.GinRespException(c, http.StatusBadRequest, fmt.Errorf("Asset params error, should be eos or eos token asset"))
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
    bal, err := b.Query.Balance(c, name, params.Asset, contract)
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
      util.GinRespException(c, http.StatusBadRequest, fmt.Errorf("Auanyify error: %d : %d", paramsQuantity.Amount, balQuantity.Amount))
      return
    }
    fromName = name
  }
  rawTxHex, err := b.Operator.RawTx(c, fromName, params.Receiptor, params.Amount, params.Memo, params.Asset)
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

  txid, err := b.Operator.BroadcastTx(c, res.HexSignedTx)
  if err != nil {
    util.GinRespException(c, http.StatusInternalServerError, err)
    return
  }

  c.JSON(http.StatusOK, gin.H {
    "status": http.StatusOK,
    "txid": txid,
  })
}
