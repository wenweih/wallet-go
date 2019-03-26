package main

import (
  "context"
  "math/big"
  "wallet-go/pkg/configure"
  "wallet-go/pkg/blockchain"
  "github.com/spf13/cobra"
  "github.com/gin-gonic/gin"
  "github.com/ethereum/go-ethereum/core/types"
  "github.com/ethereum/go-ethereum/ethclient"
)

var blockMonitor = &cobra.Command {
  Use:   "best-block",
  Short: "Best Block monitor",
  Run: func(cmd *cobra.Command, args []string) {
    switch chain {
    case "bitcoincore":
      c, err := blockchain.NewbitcoinClient()
      if err != nil {
        configure.Sugar.Fatal(err.Error())
      }
      btcClient = &blockchain.BTCRPC{Client: c}
      gin.SetMode(gin.ReleaseMode)
      r := gin.Default()
      r.GET("/btc-best-block-notify", btcBestBlockNotifyHandle)
      if err := r.Run(":3001"); err != nil {
        configure.Sugar.Fatal(err.Error())
      }
    case "ethereum":
      ethereumClient, err = ethclient.Dial(configure.Config.EthRPC)
      if err != nil {
        configure.Sugar.Fatal("Ethereum client error: ", err.Error())
      }
      defer ethereumClient.Close()
      blockCh := make(chan *types.Header)
      sub, err := ethereumClient.SubscribeNewHead(context.Background(), blockCh)
      if err != nil {
        configure.Sugar.Error(err.Error())
      }

      var (
        // maintain orderHeight and increase 1 each subscribe callback, because head.number would jump blocks
        orderHeight = new(big.Int)
      )
      for {
        select {
        case err := <-sub.Err():
          configure.Sugar.Fatal(err.Error())
        case head := <-blockCh:
          ordertmp, err := subHandle(orderHeight, head, ethereumClient)
          if err != nil {
            configure.Sugar.Error(err.Error())
          }
          orderHeight = ordertmp
        }
      }
    case "eosio":
    default:
      configure.Sugar.Fatal("Only support bitcoincore, ethereum, eosio")
    }
	},
}
