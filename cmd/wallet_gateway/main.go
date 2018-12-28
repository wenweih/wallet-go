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
  "github.com/btcsuite/btcutil"
  "wallet-transition/pkg/configure"
  "wallet-transition/pkg/blockchain"
  pb "wallet-transition/pkg/pb"
  // "github.com/shopspring/decimal"
  // "github.com/ethereum/go-ethereum/common"
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

  var (
    chainID     string
    vinAmount   int64
    unSignTxHex string
    selectedUTXOs []db.UTXO
    utxos      []db.UTXO
  )

  // raw tx
  switch asset {
  case "btc":
    fromPkScript, toPkScript, err := util.BTCWithdrawAddressValidate(*withdrawParams)
    if err != nil {
      configure.Sugar.DPanic(err.Error())
      util.GinRespException(c, http.StatusBadRequest, err)
      return
    }

    // query bitcoin current best height
    binfo, err := btcClient.Client.GetBlockChainInfo()
    if err != nil {
      configure.Sugar.DPanic("withdrawHandle err: ", err.Error())
      util.GinRespException(c, http.StatusInternalServerError, err)
      return
    }
    bheader := binfo.Headers

    feeKB, err := btcClient.Client.EstimateFee(int64(6))
    if err != nil {
      configure.Sugar.DPanic("EstimateFee: ", err.Error())
      util.GinRespException(c, http.StatusInternalServerError, err)
      return
    }

    // query utxos, which confirmate count is more than 6
    if err = sqldb.Model(subAddress).Where("height <= ? AND state = ?", bheader - 5, "original").Related(&utxos).Error; err !=nil {
      util.GinRespException(c, http.StatusNotFound, err)
      return
    }
    configure.Sugar.Info("utxos: ", utxos, " length: ", len(utxos))

    txAmount, err := btcutil.NewAmount(amount)
    if err != nil {
      configure.Sugar.DPanic("convert utxo amount(float64) to btc amount(int64 as Satoshi) error: ", err.Error())
      util.GinRespException(c, http.StatusBadRequest, err)
      return
    }

    // coin select
    selectedutxos,  selectedCoins, err := blockchain.CoinSelect(int64(bheader), txAmount, utxos)
    if err != nil {
      configure.Sugar.DPanic(err.Error())
      code := http.StatusInternalServerError
      if err.Error() == "CoinSelect error: no coin selection possible" {
        code = http.StatusBadRequest
      }
      util.GinRespException(c, code, err)
      return
    }

    for _, coin := range selectedCoins.Coins() {
      vinAmount += int64(coin.Value())
    }

    unSignTxHex = blockchain.RawBTCTx(fromPkScript, toPkScript, feeKB, txAmount, selectedCoins)
    selectedUTXOs = selectedutxos
  case "eth":
    netVersion, rawTxHex, err := ethClient.RawTx(withdrawParams.From, withdrawParams.To, amount)
    if err != nil {
      configure.Sugar.DPanic(err.Error())
      util.GinRespException(c, http.StatusBadRequest, err)
      return
    }
    chainID = *netVersion
    unSignTxHex = *rawTxHex
  }

  // sign raw tx
  grpcClient := pb.NewWalletCoreClient(rpcConn)
  ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
  defer cancel()
  res, err := grpcClient.SignTx(ctx, &pb.SignTxReq{Asset: asset.(string), From: withdrawParams.From, HexUnsignedTx: unSignTxHex, VinAmount: vinAmount, Network: chainID})
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
