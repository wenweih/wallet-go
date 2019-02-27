package main

import (
  "fmt"
  "net/http"
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
