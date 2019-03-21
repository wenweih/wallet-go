package main

import (
  "fmt"
  "context"
  "math/big"
  "wallet-go/pkg/mq"
  "wallet-go/pkg/configure"
  "wallet-go/pkg/blockchain"
  "github.com/spf13/cobra"
  "github.com/gin-gonic/gin"
  "github.com/ethereum/go-ethereum/ethclient"
  "github.com/ethereum/go-ethereum/core/types"
)

var (
  err error
  chain	string
  btcClient *blockchain.BTCRPC
  ethereumClient *ethclient.Client
  messageClient mq.IMessagingClient
)

var rootCmd = &cobra.Command {
	Use:   "ledger_monitor",
	Short: "Blockchain ledger monitor",
}

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

func execute() {
	if err := rootCmd.Execute(); err != nil {
    configure.Sugar.Fatal(fmt.Errorf("Command execute error %s", err))
	}
}

func main()  {
  execute()
}

func init()  {
  messageClient = &mq.MessagingClient{}
  messageClient.ConnectToBroker(configure.Config.MQ)
  rootCmd.AddCommand(blockMonitor)
  blockMonitor.Flags().StringVarP(&chain, "chain", "c", "", "Support bitcoincore, ethereum")
  blockMonitor.MarkFlagRequired("chain")
}
