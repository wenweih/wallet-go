package main

import (
  "fmt"
  "net/http"
  "github.com/gin-gonic/gin"
  "wallet-transition/pkg/configure"
  "wallet-transition/pkg/blockchain"
  "wallet-transition/pkg/db"
  "wallet-transition/pkg/util"
  pb "wallet-transition/pkg/pb"
)

func bitcoincoreWalletHandle(c *gin.Context) {
  asset, _ := c.Get("asset")
  chain := configure.ChainAssets[asset.(string)]
  if chain == blockchain.Bitcoin {
    res, err := grpcClient.BitcoinWallet(c, &pb.BitcoinWalletReq{Mode: bitcoinnet.Net.String()})
    if err != nil {
      util.GinRespException(c, http.StatusInternalServerError, err)
      return
    }
    address := res.Address
    if err := sqldb.Create(&db.SubAddress{Address: address, Asset: blockchain.Bitcoin}).Error; err != nil {
      util.GinRespException(c, http.StatusInternalServerError, err)
      return
    }
    c.JSON(http.StatusOK, gin.H {
      "status": http.StatusOK,
      "address": address,
    })
  }else {
    util.GinRespException(c, http.StatusBadRequest, fmt.Errorf("%s is't bitcoin asset", asset.(string)))
    return
  }
}
