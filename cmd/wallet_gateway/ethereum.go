package main

import (
  "fmt"
  "errors"
  "math/big"
  "net/http"
  "encoding/json"
  "github.com/gin-gonic/gin"
  "wallet-transition/pkg/util"
  "wallet-transition/pkg/db"
  "wallet-transition/pkg/blockchain"
  "wallet-transition/pkg/configure"
  empty "github.com/golang/protobuf/ptypes/empty"
)

func ethereumWalletHandle(c *gin.Context) {
  asset, _ := c.Get("asset")
  chain := configure.ChainAssets[asset.(string)]
  if chain == blockchain.Ethereum {
    res, err := grpcClient.EthereumWallet(c, &empty.Empty{})
    if err != nil {
      util.GinRespException(c, http.StatusInternalServerError, err)
      return
    }
    address := res.Address
    if err := sqldb.Create(&db.SubAddress{Address: address, Asset: blockchain.Ethereum}).Error; err != nil {
      util.GinRespException(c, http.StatusInternalServerError, err)
      return
    }
    c.JSON(http.StatusOK, gin.H {
      "status": http.StatusOK,
      "address": address,
    })
  }else {
    util.GinRespException(c, http.StatusBadRequest, fmt.Errorf("%s is't ethereum asset", asset.(string)))
    return
  }
}

func ethereumBalanceHandle(c *gin.Context) {
  asset, _ := c.Get("asset")
  detailParams, _ := c.Get("detail")

  if configure.ChainAssets[asset.(string)] != blockchain.Ethereum {
    util.GinRespException(c, http.StatusBadRequest, fmt.Errorf("%s is't Ethereum asset", asset.(string)))
    return
  }

  var balanceParams util.BalanceParams
  if err := json.Unmarshal(detailParams.([]byte), &balanceParams); err != nil {
    util.GinRespException(c, http.StatusInternalServerError, err)
    return
  }

  if balanceParams.Address == "" {
    util.GinRespException(c, http.StatusInternalServerError, errors.New("address param is required"))
    return
  }

  chain := blockchain.EthereumChain{Client: ethereumClient}
  b := blockchain.NewBlockchain(nil, nil, chain)
  balance, err := b.Query.Balance(balanceParams.Address, balanceParams.Asset, "")
  if err != nil {
    util.GinRespException(c, http.StatusInternalServerError, err)
    return
  }
  amount, ok := new(big.Int).SetString(balance, 10)
  if !ok {
    util.GinRespException(c, http.StatusInternalServerError, errors.New("Set amount error"))
    return
  }
  c.JSON(http.StatusOK, gin.H {
    "status": http.StatusOK,
    "balance": util.ToEther(amount).String(),
  })
}
