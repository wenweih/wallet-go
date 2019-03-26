package main

import (
  "fmt"
  "wallet-go/pkg/mq"
  "wallet-go/pkg/configure"
  "wallet-go/pkg/blockchain"
  "github.com/spf13/cobra"
  "github.com/ethereum/go-ethereum/ethclient"
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

func main()  {
  if err := rootCmd.Execute(); err != nil {
    configure.Sugar.Fatal(fmt.Errorf("Command execute error %s", err))
  }
}

func init()  {
  messageClient = &mq.MessagingClient{}
  messageClient.ConnectToBroker(configure.Config.MQ)
  rootCmd.AddCommand(blockMonitor)
  blockMonitor.Flags().StringVarP(&chain, "chain", "c", "", "Support bitcoincore, ethereum")
  blockMonitor.MarkFlagRequired("chain")
}
