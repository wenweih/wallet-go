package main

import (
  "time"
  "errors"
  "context"
  "net/http"
  "reflect"
  "google.golang.org/grpc"
  "github.com/gin-gonic/gin"
  "wallet-transition/pkg/db"
  "wallet-transition/pkg/util"
  pb "wallet-transition/pkg/pb"
  "wallet-transition/pkg/configure"
)

var (
  sqldb   *db.GormDB
  rpcConn *grpc.ClientConn
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

  if withdrawParams.Amount <= 0 {
    util.GinRespException(c, http.StatusBadRequest, errors.New("amount can't be empty and less than 0"))
    return
  }
  if withdrawParams.From == "" || withdrawParams.To == "" {
    util.GinRespException(c, http.StatusBadRequest, errors.New("from or to params can't be empty"))
    return
  }
  switch asset {
  case "btc":
    configure.Sugar.Info("btc")
  case "eth":
    configure.Sugar.Info("eth")
  }
}
