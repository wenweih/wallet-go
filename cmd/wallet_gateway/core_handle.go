package main

import (
  "time"
  "errors"
  "strconv"
  "context"
  "net/http"
  "github.com/gin-gonic/gin"
  "wallet-transition/pkg/db"
  "wallet-transition/pkg/util"
  "wallet-transition/pkg/configure"
  "wallet-transition/pkg/blockchain"
  pb "wallet-transition/pkg/pb"
)

func addressHandle(c *gin.Context) {
  asset, _ := c.Get("asset")
  grpcClient := pb.NewWalletCoreClient(rpcConn)
  ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
  defer cancel()
  res, err := grpcClient.Address(ctx, &pb.AddressReq{Asset: asset.(string)})
  if err != nil {
    util.GinRespException(c, http.StatusInternalServerError, err)
    return
  }

  if err := sqldb.Create(&db.SubAddress{Address: res.Address, Asset: asset.(string)}).Error; err != nil {
    util.GinRespException(c, http.StatusInternalServerError, err)
    return
  }

  c.JSON(http.StatusOK, gin.H {
    "status": http.StatusOK,
    "address": res.Address,
  })
}

func withdrawHandle(c *gin.Context)  {
  asset, _ := c.Get("asset")
  detailParams, _ := c.Get("detail")
  withdrawParams, subAddress, err := util.WithdrawParamsH(detailParams, asset.(string), sqldb)
  if err != nil {
    util.GinRespException(c, http.StatusBadRequest, err)
    return
  }

  // params
  amount, err := strconv.ParseFloat(withdrawParams.Amount, 64)
  if err != nil {
    util.GinRespException(c, http.StatusBadRequest, errors.New("amount can't be empty and less than 0"))
    return
  }

  // raw tx
  unSignTxHex, chainID, vinAmount, selectedUTXOs, httpStatus, err := blockchain.RawTx(withdrawParams.From, withdrawParams.To, asset.(string), amount, subAddress, btcClient, ethClient, sqldb)
  if err != nil {
    configure.Sugar.DPanic(err.Error())
    util.GinRespException(c, httpStatus, err)
    return
  }

  // sign raw tx
  grpcClient := pb.NewWalletCoreClient(rpcConn)
  ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
  defer cancel()
  res, err := grpcClient.SignTx(ctx, &pb.SignTxReq{Asset: asset.(string), From: withdrawParams.From, HexUnsignedTx: *unSignTxHex, VinAmount: *vinAmount, Network: *chainID})
  if err != nil {
    util.GinRespException(c, http.StatusInternalServerError, err)
    return
  }

  // send signed tx
  txid, httpStatus, err := blockchain.SendTx(asset.(string), res.HexSignedTx, selectedUTXOs, btcClient, ethClient, sqldb)
  if err != nil {
    configure.Sugar.DPanic(err.Error())
    util.GinRespException(c, httpStatus, err)
    return
  }

  c.JSON(http.StatusOK, gin.H {
    "status": http.StatusOK,
    "txid": *txid,
  })
}
