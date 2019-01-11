package main

import (
  "strconv"
  "net/http"
  "github.com/gin-gonic/gin"
  "wallet-transition/pkg/util"
  "wallet-transition/pkg/configure"
  "wallet-transition/pkg/blockchain"
  "github.com/btcsuite/btcutil"
  pb "wallet-transition/pkg/pb"
)

func addressHandle(c *gin.Context) {
  asset, _ := c.Get("asset")
  address, err := genAddress(c, asset.(string))
  if err != nil {
    util.GinRespException(c, http.StatusInternalServerError, err)
    return
  }

  c.JSON(http.StatusOK, gin.H {
    "status": http.StatusOK,
    "address": *address,
  })
}

func withdrawHandle(c *gin.Context)  {
  assetParams, _ := c.Get("asset")
  detailParams, _ := c.Get("detail")

  keys := make([]string, len(configure.Config.ETHToken))
  for k := range configure.Config.ETHToken {
    keys = append(keys, k)
  }
  asset := assetParams.(string)
  if util.Contain(asset, keys) {
    asset = "eth"
  }

  withdrawParams, subAddress, err := util.WithdrawParamsH(detailParams, asset, sqldb)
  if err != nil {
    util.GinRespException(c, http.StatusBadRequest, err)
    return
  }

  // params
  amount, err := strconv.ParseFloat(withdrawParams.Amount, 64)
  if err != nil {
    util.GinRespException(c, http.StatusBadRequest, err)
    return
  }

  // raw tx
  unSignTxHex, chainID, vinAmount, selectedUTXOs, httpStatus, err := blockchain.RawTx(c, withdrawParams.From,
    withdrawParams.To, assetParams.(string), amount, subAddress, btcClient, ethClient, sqldb)
  if err != nil {
    configure.Sugar.DPanic(err.Error())
    util.GinRespException(c, httpStatus, err)
    return
  }

  // sign raw tx
  res, err := grpcClient.SignTx(c, &pb.SignTxReq{Asset: asset,
    From: withdrawParams.From,
    HexUnsignedTx: *unSignTxHex,
    VinAmount: *vinAmount,
    Network: *chainID})
  if err != nil {
    util.GinRespException(c, http.StatusInternalServerError, err)
    return
  }

  configure.Sugar.Info("signeddddd: ", res.HexSignedTx)

  // send signed tx
  txid, httpStatus, err := blockchain.SendTx(c, asset, res.HexSignedTx, selectedUTXOs, btcClient, ethClient, sqldb)
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

func sendToAddress(c *gin.Context)  {
  assetParams, _ := c.Get("asset")
  detailParams, _ := c.Get("detail")

  sendToAddressParams, err := util.SendToAddressParamsH(detailParams)
  if err != nil {
    util.GinRespException(c, http.StatusBadRequest, err)
    return
  }

  amount, err := strconv.ParseFloat(sendToAddressParams.Amount, 64)
  if err != nil {
    util.GinRespException(c, http.StatusBadRequest, err)
    return
  }

  txAmount, _ := btcutil.NewAmount(amount)
  funbackAddress, err := genAddress(c, "btc")
  if err != nil {
    util.GinRespException(c, http.StatusBadRequest, err)
    return
  }
  vAmount, selectedutxos, rawTxHex, httpStatus, err := btcClient.RawSendToAddressTx(txAmount, *funbackAddress, sendToAddressParams.To, sqldb)
  if err != nil {
    configure.Sugar.DPanic(err.Error())
    util.GinRespException(c, httpStatus, err)
    return
  }

  var pOutPoints []*pb.SendToAddressReq_PreviousOutPoint
  for _, utxo := range selectedutxos {
    configure.Sugar.Info("xxxx: ", utxo.ID)
    pOutPoints = append(pOutPoints, &pb.SendToAddressReq_PreviousOutPoint{Txid: utxo.Txid, Index: utxo.VoutIndex, Address: utxo.SubAddress.Address})
  }
  res, err := grpcClient.SendToAddressSignBTC(c, &pb.SendToAddressReq{VinAmount: *vAmount, HexUnsignedTx: *rawTxHex, Utxo: pOutPoints})
  if err != nil {
    configure.Sugar.Info("SendToAddressSignBTC error: ", err.Error())
  }
  configure.Sugar.Info("xxxxx: ", res.HexSignedTx)
  // send signed tx
  txid, httpStatus, err := blockchain.SendTx(c, assetParams.(string), res.HexSignedTx, selectedutxos, btcClient, ethClient, sqldb)
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
