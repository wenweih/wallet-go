package main

import (
  "google.golang.org/grpc"
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
  omniClient *blockchain.BTCRPC
  ethClient *blockchain.ETHRPC
  grpcClient pb.WalletCoreClient
)

func main() {
  var err error
  // sqldb, err = db.NewSqlite()
  sqldb, err = db.NewMySQL()
  if err != nil {
    configure.Sugar.Fatal(err.Error())
  }
  defer sqldb.Close()

  rpcConn, err = grpc.Dial(configure.Config.WalletCoreRPCURL, grpc.WithInsecure())
  if err != nil {
    configure.Sugar.Fatal("fail to connect grpc server")
  }
  defer rpcConn.Close()
  grpcClient = pb.NewWalletCoreClient(rpcConn)

  omniClient = &blockchain.BTCRPC{Client: blockchain.NewOmnicoreClient()}
  btcClient = &blockchain.BTCRPC{Client: blockchain.NewbitcoinClient()}
  ethClient, err = blockchain.NewEthClient()
  if err != nil {
    configure.Sugar.Fatal("Ethereum client error: ", err.Error())
  }

  balance, _ := omniClient.GetOmniBalance("mqb6duu66oFYr257DJKp2KGm7KESCeb4fq", 2147483652)
  configure.Sugar.Info("balance: ", balance)

  r := util.GinEngine()
  r.POST("/address", addressHandle)
  r.POST("/send", withdrawHandle)
  r.POST("/sendtoaddress", sendToAddress)

  r.GET("/tx", txHandle)
  r.GET("/block", blockHandle)
  r.GET("/ethereum_balance", ethereumBalanceHandle)
  r.GET("/omni_balance", omniBalanceHandle)
  r.GET("/address_validator", addressValidator)
  r.GET("/best_block", bestBlock)
  if err := r.Run(":8000"); err != nil {
    configure.Sugar.Fatal(err.Error())
  }
}
