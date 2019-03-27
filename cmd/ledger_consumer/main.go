package main

import (
  "fmt"
  "github.com/spf13/cobra"
  "wallet-go/pkg/configure"
)

var (
  chain	string
)

var rootCmd = &cobra.Command {
	Use:   "ledger_consumer",
	Short: "Blockchain ledger consumer. maintain UTXO or deposit notify",
}

func main()  {
  if err := rootCmd.Execute(); err != nil {
    configure.Sugar.Fatal(fmt.Errorf("Command execute error %s", err))
	}
}

func init()  {
  rootCmd.AddCommand(UTXO)
  UTXO.Flags().StringVarP(&chain, "chain", "c", "", "Support bitcoincore")
  UTXO.MarkFlagRequired("chain")
}
