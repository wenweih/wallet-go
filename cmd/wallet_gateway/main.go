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
  "wallet-transition/pkg/util"
  pb "wallet-transition/pkg/pb"
  "wallet-transition/pkg/configure"
)

func main()  {
  pubBytes, err := ioutil.ReadFile(strings.Join([]string{configure.HomeDir(), "wallet_pub.pem"}, "/"))
  if err != nil {
    configure.Sugar.Fatal(err.Error())
  }
  rsaPub := util.BytesToPublicKey(pubBytes)

  params := util.AddressParams {
    Asset : "btc",
  }
  paramsBytes, err := json.Marshal(params)
  if err != nil {
    configure.Sugar.Warn("json Marshal error: ", err.Error())
  }

  encryptAccountPriv := util.EncryptWithPublicKey(paramsBytes, rsaPub)
  configure.Sugar.Info(hex.EncodeToString(encryptAccountPriv))

  r := util.GinEngine()
  r.POST("/address", addressHandle)
  if err := r.Run(":3000"); err != nil {
    configure.Sugar.Fatal(err.Error())
  }
}

func addressHandle(c *gin.Context) {
  paramsByte, exist := c.Get("params")
  if !exist {
    util.GinRespException(c, http.StatusInternalServerError, errors.New("paramsByte not exist"))
    return
  }

  var params util.AddressParams
  if err := json.Unmarshal(paramsByte.([]byte), &params); err != nil {
    util.GinRespException(c, http.StatusInternalServerError, errors.New("Unmarshal params error"))
    return
  }
  configure.Sugar.Info("xxx: ", params)

  conn, err := grpc.Dial("127.0.0.1:50051", grpc.WithInsecure())
  if err != nil {
    configure.Sugar.Fatal("fail to connect grpc server")
  }
  defer conn.Close()
  grpcClient := pb.NewWalletCoreClient(conn)
  ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
  defer cancel()
  res, err := grpcClient.Address(ctx, &pb.AddressReq{Asset: "btc"})
  if err != nil {
    util.GinRespException(c, http.StatusInternalServerError, err)
    return
  }

  c.JSON(http.StatusOK, gin.H {
    "status": http.StatusOK,
    "address": res.Address,
  })
}
