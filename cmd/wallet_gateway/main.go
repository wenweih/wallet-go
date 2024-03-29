package main

import (
  "flag"
  "google.golang.org/grpc"
  "wallet-go/pkg/db"
  "wallet-go/pkg/util"
  "wallet-go/pkg/configure"
  "wallet-go/pkg/blockchain"
  pb "wallet-go/pkg/pb"
  "github.com/btcsuite/btcd/chaincfg"
  "github.com/eoscanada/eos-go"
  "github.com/ethereum/go-ethereum/ethclient"
  "github.com/btcsuite/btcd/rpcclient"
)

var (
  sqldb   *db.GormDB
  rpcConn *grpc.ClientConn
  btcClient *blockchain.BTCRPC
  bitcoinClient *rpcclient.Client
  omniClient *rpcclient.Client
  ethereumClient *ethclient.Client
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

  bitcoinClient, err = blockchain.NewbitcoinClient()
  if err != nil {
    configure.Sugar.Fatal(err.Error())
  }
  omniClient, err = blockchain.NewOmnicoreClient()
  if err != nil {
    configure.Sugar.Fatal(err.Error())
  }

  ethereumClient, err = ethclient.Dial(configure.Config.EthRPC)
  if err != nil {
    configure.Sugar.Fatal("Ethereum client error: ", err.Error())
  }

  eosClient = eos.New(configure.Config.EOSIORPC)

  r := util.GinEngine()

  r.POST("/eosio/wallet", eosioWalletHandle)
  r.POST("/eosio/tx", eosiotxHandle)
  r.GET("/eosio/balance", eosioBalanceHandle)

  r.POST("/bitcoincore/wallet", bitcoincoreWalletHandle)
  r.POST("/bitcoincore/tx", bitcoincoreWithdrawHandle)

  r.POST("/ethereum/wallet", ethereumWalletHandle)
  r.GET("/ethereum/balance", ethereumBalanceHandle)
  r.POST("/ethereum/tx", ethereumWithdrawHandle)

  r.GET("/omnicore/balance", omniBalanceHandle)

  r.GET("/tx", txHandle)
  r.GET("/block", blockHandle)
  r.GET("/address_validator", addressValidator)
  r.GET("/best_block", bestBlock)
  if err := r.Run(":8000"); err != nil {
    configure.Sugar.Fatal(err.Error())
  }
}
