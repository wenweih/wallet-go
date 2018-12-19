package main

import (
  "math/big"
  "time"
  "errors"
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
  "github.com/shopspring/decimal"
  "github.com/ethereum/go-ethereum/common"
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

  rpcConn, err = grpc.Dial("127.0.0.1:50051", grpc.WithInsecure())
  if err != nil {
    configure.Sugar.Fatal("fail to connect grpc server")
  }
  // defer rpcConn.Close()
  defer sqldb.Close()

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
  if err := sqldb.First(&subAddress, "address = ? AND asset = ?", withdrawParams.From, asset).Error; err !=nil && err.Error() == "record not found" {
    util.GinRespException(c, http.StatusNotFound, errors.New(strings.Join([]string{asset.(string), " ", withdrawParams.From, " not found in database"}, "")))
    return
  }else if err != nil {
    util.GinRespException(c, http.StatusNotFound, err)
    return
  }

  var (
    vinAmount   int64
    unSignTxHex string
  )

  switch asset {
  case "btc":
    fromPkScript, toPkScript, err := util.BTCWithdrawAddressValidate(withdrawParams)
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
    if err = sqldb.Model(&subAddress).Where("height <= ? AND used = ?", bheader - 4, false).Related(&utxos).Error; err !=nil {
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

    configure.Sugar.Info("selectedUTXOs: ", selectedUTXOs, " length: ", len(selectedUTXOs))
    configure.Sugar.Info("selectedCoins: ", selectedCoins, " length: ", len(selectedCoins.Coins()))

    unSignTxHex = blockchain.RawBTCTx(fromPkScript, toPkScript, feeKB, txAmount, selectedCoins)
  case "eth":
    if !common.IsHexAddress(withdrawParams.To) {
      err := errors.New(strings.Join([]string{"To: ", withdrawParams.To, " isn't valid ethereum address"}, ""))
      configure.Sugar.DPanic(err.Error())
      util.GinRespException(c, http.StatusBadRequest, err)
      return
    }

    var (
      txFee = new(big.Int)
    )
    gasLimit := uint64(21000) // in units
    balance, nonce, gasPrice, err := ethClient.GetBalanceAndPendingNonceAtAndGasPrice(context.Background(), subAddress.Address)
    if err != nil {
      configure.Sugar.DPanic(err.Error())
      util.GinRespException(c, http.StatusInternalServerError, err)
      return
    }
    etherToWei := decimal.NewFromBigInt(big.NewInt(1000000000000000000), 0)

    balanceDecimal, _ := decimal.NewFromString(balance.String())
    transferAmount := decimal.NewFromFloat(withdrawParams.Amount)
    transferAmount = transferAmount.Mul(etherToWei)
    txFee = txFee.Mul(gasPrice, big.NewInt(int64(gasLimit)))
    feeDecimal, _ := decimal.NewFromString(txFee.String())
    totalCost := transferAmount.Add(feeDecimal)
    if !totalCost.LessThanOrEqual(balanceDecimal) {
      err = errors.New(strings.Join([]string{"Account: ", withdrawParams.From, " balance is not enough ", balanceDecimal.String(), ":", totalCost.String()}, ""))
      configure.Sugar.DPanic(err.Error())
      util.GinRespException(c, http.StatusBadRequest, err)
      return
    }

    amount, _ := new(big.Int).SetString(transferAmount.String(), 10)
    rawTxHex, _, err := blockchain.CreateRawETHTx(*nonce, amount, gasPrice, withdrawParams.From, withdrawParams.To)
    if err != nil {
      err := errors.New(strings.Join([]string{"To: ", withdrawParams.To, " isn't valid ethereum address"}, ""))
      configure.Sugar.DPanic(err.Error())
      util.GinRespException(c, http.StatusBadRequest, err)
      return
    }
    unSignTxHex = *rawTxHex
  }

  grpcClient := pb.NewWalletCoreClient(rpcConn)
  ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
  defer cancel()
  res, err := grpcClient.SignTx(ctx, &pb.SignTxReq{Asset: asset.(string), From: withdrawParams.From, HexUnsignedTx: unSignTxHex, VinAmount: vinAmount})
  if err != nil {
    util.GinRespException(c, http.StatusInternalServerError, err)
    return
  }

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
  case "eth":
    tx, err := blockchain.DecodeETHTx(res.HexSignedTx)
    if err != nil {
      e := errors.New(strings.Join([]string{"Decode signed tx error", err.Error()}, ":"))
      configure.Sugar.DPanic(e.Error())
      util.GinRespException(c, http.StatusInternalServerError, e)
      return
    }
    if err := ethClient.Client.SendTransaction(context.Background(), tx); err != nil {
      e := errors.New(strings.Join([]string{"Ethereum SendTransactionsigned tx error", err.Error()}, ":"))
      configure.Sugar.DPanic(e.Error())
      util.GinRespException(c, http.StatusInternalServerError, e)
      return
    }
    txid = tx.Hash().String()
  }

  c.JSON(http.StatusOK, gin.H {
    "status": http.StatusOK,
    "txid": txid,
  })
}
