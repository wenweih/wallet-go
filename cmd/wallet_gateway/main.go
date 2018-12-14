package main

import (
  "time"
  "errors"
  "bytes"
  "strings"
  "context"
  "net/http"
  "reflect"
  "encoding/hex"
  "google.golang.org/grpc"
  "github.com/gin-gonic/gin"
  "wallet-transition/pkg/db"
  "wallet-transition/pkg/util"
  "github.com/btcsuite/btcutil"
  "github.com/btcsuite/btcd/wire"
  "wallet-transition/pkg/configure"
  "wallet-transition/pkg/blockchain"
  "github.com/btcsuite/btcd/chaincfg"
  "github.com/btcsuite/btcd/txscript"
  "github.com/btcsuite/btcutil/coinset"
  pb "wallet-transition/pkg/pb"
)

var (
  sqldb   *db.GormDB
  rpcConn *grpc.ClientConn
  btcClient *blockchain.BTCRPC
)

func main() {
  var err error
  sqldb, err = db.NewSqlite()
  if err != nil {
    configure.Sugar.Fatal(err.Error())
  }
  defer sqldb.Close()

  rpcConn, err = grpc.Dial("127.0.0.1:50051", grpc.WithInsecure())
  if err != nil {
    configure.Sugar.Fatal("fail to connect grpc server")
  }
  defer rpcConn.Close()
  defer sqldb.Close()

  btcClient = &blockchain.BTCRPC{Client: blockchain.NewbitcoinClient()}

  r := util.GinEngine()
  r.POST("/address", addressHandle)
  r.POST("/withdraw", withdrawHandle)
  if err := r.Run(":3000"); err != nil {
    configure.Sugar.Fatal(err.Error())
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
  params := reflect.ValueOf(detailParams)
  withdrawParams := util.WithdrawParams{}
  if params.Kind() == reflect.Map {
    for _, key := range params.MapKeys() {
      switch key.Interface() {
      case "from":
        withdrawParams.From = params.MapIndex(key).Interface().(string)
      case "to":
        withdrawParams.To = params.MapIndex(key).Interface().(string)
      case "amount":
        withdrawParams.Amount = params.MapIndex(key).Interface().(float64)
      }
    }
  }else {
    util.GinRespException(c, http.StatusBadRequest, errors.New("detail params error"))
    return
  }

  // params
  if withdrawParams.Amount <= 0 {
    util.GinRespException(c, http.StatusBadRequest, errors.New("amount can't be empty and less than 0"))
    return
  }
  if withdrawParams.From == "" || withdrawParams.To == "" {
    util.GinRespException(c, http.StatusBadRequest, errors.New("from or to params can't be empty"))
    return
  }

  var (
    subAddress db.SubAddress
    utxos      []db.UTXO
  )
  // query from address
  if err := sqldb.Where("address = ? AND asset = ?", withdrawParams.From, asset).First(&subAddress).Error; err !=nil && err.Error() == "record not found" {
    util.GinRespException(c, http.StatusNotFound, errors.New(strings.Join([]string{withdrawParams.From, " not found in database"}, "")))
    return
  }else if err != nil {
    util.GinRespException(c, http.StatusNotFound, err)
    return
  }

  switch asset {
  case "btc":
    toAddress, err := btcutil.DecodeAddress(withdrawParams.To, &chaincfg.RegressionNetParams)
    if err != nil {
      e := errors.New(strings.Join([]string{"To address illegal", err.Error()}, ":"))
      configure.Sugar.DPanic(e.Error())
      util.GinRespException(c, http.StatusBadRequest, e)
      return
    }

    fromAddress, err := btcutil.DecodeAddress(withdrawParams.From, &chaincfg.RegressionNetParams)
    if err != nil {
      e := errors.New(strings.Join([]string{"From address address illegal", err.Error()}, ":"))
      configure.Sugar.DPanic(e.Error())
      util.GinRespException(c, http.StatusBadRequest, e)
      return
    }

    toPkScript, err := txscript.PayToAddrScript(toAddress)
    if err != nil {
      e := errors.New(strings.Join([]string{"to address PayToAddrScript error", err.Error()}, ":"))
      configure.Sugar.DPanic(e.Error())
      util.GinRespException(c, http.StatusInternalServerError, e)
      return
    }

    fromPkScript, err := txscript.PayToAddrScript(fromAddress)
    if err != nil {
      e := errors.New(strings.Join([]string{"from address PayToAddrScript error", err.Error()}, ":"))
      configure.Sugar.DPanic(e.Error())
      util.GinRespException(c, http.StatusInternalServerError, e)
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

    // query utxos, which confirmate count is more than 6
    if err = sqldb.Model(&subAddress).Where("height <= ?", bheader - 4).Related(&utxos).Error; err !=nil {
      util.GinRespException(c, http.StatusNotFound, err)
      return
    }
    configure.Sugar.Info("utxos: ", utxos, " length: ", len(utxos))

    txAmount, err := btcutil.NewAmount(withdrawParams.Amount)
    if err != nil {
      configure.Sugar.DPanic("convert utxo amount(float64) to btc amount(int64 as Satoshi) error: ", err.Error())
      util.GinRespException(c, http.StatusBadRequest, err)
      return
    }

    // coin select
    selectedUTXOs,  selectedCoins, err := blockchain.CoinSelect(int64(bheader), txAmount, utxos)
    if err != nil {
      configure.Sugar.DPanic(err.Error())
      util.GinRespException(c, http.StatusInternalServerError, err)
      return
    }

    configure.Sugar.Info("selectedUTXOs: ", selectedUTXOs, " length: ", len(selectedUTXOs))
    configure.Sugar.Info("selectedCoins: ", selectedCoins, " length: ", len(selectedCoins.Coins()))

    msgTx := coinset.NewMsgTxWithInputCoins(wire.TxVersion, selectedCoins)

    var vinAmount int64
    for _, coin := range selectedCoins.Coins() {
      vinAmount += int64(coin.Value())
    }

    txOutTo := wire.NewTxOut(int64(txAmount), toPkScript)
    txOutReBack := wire.NewTxOut((vinAmount-int64(txAmount)), fromPkScript) // todo: sub tx fee
    msgTx.AddTxOut(txOutTo)
    msgTx.AddTxOut(txOutReBack)

    buf := bytes.NewBuffer(make([]byte, 0, msgTx.SerializeSize()))
    msgTx.Serialize(buf)
    txHex := hex.EncodeToString(buf.Bytes())
    configure.Sugar.Info("txHex: ", txHex)
  case "eth":
    configure.Sugar.Info("eth")
  }
}
