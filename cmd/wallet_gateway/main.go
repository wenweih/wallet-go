package main

import (
  "time"
  "context"
  "errors"
  "strings"
  "io/ioutil"
  "net/http"
  "encoding/hex"
	"encoding/json"
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

  pubBytes, err := ioutil.ReadFile(strings.Join([]string{configure.HomeDir(), "wallet_pub.pem"}, "/"))
  if err != nil {
    configure.Sugar.Fatal(err.Error())
  }
  rsaPub := util.BytesToPublicKey(pubBytes)

  params := util.AuthParams {
    Asset : "eth",
  }
  paramsBytes, err := json.Marshal(params)
  if err != nil {
    configure.Sugar.Warn("json Marshal error: ", err.Error())
  }

  encryptAccountPriv := util.EncryptWithPublicKey(paramsBytes, rsaPub)
  configure.Sugar.Info(hex.EncodeToString(encryptAccountPriv))

  defer sqldb.Close()

  r := util.GinEngine()
  r.POST("/address", addressHandle)
  r.POST("/withdraw", withdrawHandle)
  if err := r.Run(":3000"); err != nil {
    configure.Sugar.Fatal(err.Error())
  }
}

func addressHandle(c *gin.Context) {
  asset, exist := c.Get("asset")
  if !exist {
    util.GinRespException(c, http.StatusInternalServerError, errors.New("paramsByte not exist"))
    return
  }

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

}
