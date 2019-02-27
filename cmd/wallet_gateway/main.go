package main

import (
  "flag"
  "google.golang.org/grpc"
  "wallet-transition/pkg/db"
  "wallet-transition/pkg/util"
  "wallet-transition/pkg/configure"
  "wallet-transition/pkg/blockchain"
  pb "wallet-transition/pkg/pb"
  "github.com/btcsuite/btcd/chaincfg"
  "github.com/eoscanada/eos-go"
)

var (
  sqldb   *db.GormDB
  rpcConn *grpc.ClientConn
  btcClient *blockchain.BTCRPC
  omniClient *blockchain.BTCRPC
  ethClient *blockchain.ETHRPC
  eosClient *eos.API
  grpcClient pb.WalletCoreClient
  bitcoinnet *chaincfg.Params
)

func main() {
  var (
    err error
    bitcoinmode string
  )

  flag.StringVar(&bitcoinmode, "bitcoinmode", "mainnet", "btc base chain mode: testnet, regtest or mainnet")
  flag.Parse()
  bitcoinnet, err = blockchain.BitcoinNet(bitcoinmode)
  if err != nil {
    configure.Sugar.Fatal(err.Error())
  }
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

  eosClient = eos.New(configure.Config.EOSIORPC)

  r := util.GinEngine()
  r.POST("/send", withdrawHandle)
  r.POST("/sendtoaddress", sendToAddress)

  r.POST("/eosio/wallet", eosioWalletHandle)
  r.POST("/eosio/tx", eosiotxHandle)
  r.GET("/eosio/balance", eosioBalanceHandle)

  r.POST("/bitcoincore/wallet", bitcoincoreWalletHandle)

  r.POST("/ethereum/wallet", ethereumWalletHandle)
  r.GET("/ethereum/balance", ethereumBalanceHandle)

  r.GET("/tx", txHandle)
  r.GET("/block", blockHandle)
  r.GET("/omnicore_balance", omniBalanceHandle)
  r.GET("/address_validator", addressValidator)
  r.GET("/best_block", bestBlock)
  if err := r.Run(":8000"); err != nil {
    configure.Sugar.Fatal(err.Error())
  }
}
