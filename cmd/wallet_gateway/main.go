package main

import (
  // "math/big"
  "time"
  "errors"
  "strconv"
  "strings"
  "context"
  "net/http"
  "reflect"
  "google.golang.org/grpc"
  "github.com/gin-gonic/gin"
  "wallet-transition/pkg/db"
  "wallet-transition/pkg/util"
  "wallet-transition/pkg/configure"
  "wallet-transition/pkg/blockchain"
  pb "wallet-transition/pkg/pb"
)

var (
  sqldb   *db.GormDB
  rpcConn *grpc.ClientConn
  btcClient *blockchain.BTCRPC
  ethClient *blockchain.ETHRPC
)

func main() {
  var err error
  sqldb, err = db.NewSqlite()
  if err != nil {
    configure.Sugar.Fatal(err.Error())
  }
  defer sqldb.Close()

  rpcConn, err = grpc.Dial(configure.Config.WalletCoreRPCURL, grpc.WithInsecure())
  if err != nil {
    configure.Sugar.Fatal("fail to connect grpc server")
  }
  defer rpcConn.Close()

  btcClient = &blockchain.BTCRPC{Client: blockchain.NewbitcoinClient()}
  ethClient, err = blockchain.NewEthClient()
  if err != nil {
    configure.Sugar.Fatal("Ethereum client error: ", err.Error())
  }

  r := util.GinEngine()
  r.POST("/address", addressHandle)
  r.POST("/withdraw", withdrawHandle)
  r.POST("/block", blockHandle)
  if err := r.Run(":3000"); err != nil {
    configure.Sugar.Fatal(err.Error())
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
        blockParams.Height = params.MapIndex(key).Interface().(int64)
      }
    }
  }else {
    util.GinRespException(c, http.StatusBadRequest, errors.New("detail params error"))
    return
  }
  switch asset.(string) {
  case "btc":
    block, err := btcClient.GetBlock(blockParams.Height)
    if err !=nil {
      util.GinRespException(c, http.StatusInternalServerError, err)
      return
    }
    c.JSON(http.StatusOK, gin.H {
      "status": http.StatusOK,
      "block": block,
    })
    return
  }
}

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
  txid := ""
  switch asset.(string) {
  case "btc":
    tx, err :=blockchain.DecodeBtcTxHex(res.HexSignedTx)
    if err != nil {
      e := errors.New(strings.Join([]string{"Decode signed tx error", err.Error()}, ":"))
      configure.Sugar.DPanic(e.Error())
      util.GinRespException(c, http.StatusInternalServerError, e)
      return
    }

    txHash, err := btcClient.Client.SendRawTransaction(tx.MsgTx(), false)
    if err != nil {
      e := errors.New(strings.Join([]string{"Bitcoin SendRawTransaction signed tx error", err.Error()}, ":"))
      configure.Sugar.DPanic(e.Error())
      util.GinRespException(c, http.StatusInternalServerError, e)
      return
    }
    txid = txHash.String()
    ts := sqldb.Begin()
    for _, dbutxo := range selectedUTXOs {
      ts.Model(&dbutxo).Updates(map[string]interface{}{"used_by": txid, "state": "selected"})
    }
    if err := ts.Commit().Error; err != nil {
      e := errors.New(strings.Join([]string{"update selected utxos error", err.Error()}, ":"))
      configure.Sugar.DPanic(e.Error())
      util.GinRespException(c, http.StatusInternalServerError, e)
      return
    }
  case "eth":
    ethTxid, err := ethClient.SendTx(res.HexSignedTx)
    if err != nil {
      configure.Sugar.DPanic(err.Error())
      util.GinRespException(c, http.StatusInternalServerError, err)
      return
    }
    txid = *ethTxid
  }

  c.JSON(http.StatusOK, gin.H {
    "status": http.StatusOK,
    "txid": txid,
  })
}
