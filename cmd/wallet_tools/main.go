package main

import (
	"github.com/spf13/cobra"
	"wallet-transition/pkg/configure"
	"wallet-transition/pkg/blockchain"
)

var rootCmd = &cobra.Command{
	Use:   "wallet-transition-tool",
	Short: "Commandline to for anbi exchange wallet module",
}

func execute() {
	if err := rootCmd.Execute(); err != nil {
		configure.Sugar.Fatal("Command execute error: ", err.Error())
	}
}

var migrateWallet = &cobra.Command{
	Use:   "migrate",
	Short: "Migrate wallet from blockchain node",
	Run: func(cmd *cobra.Command, args []string) {
		btcClient := blockchain.BitcoinClientAlias{blockchain.NewbitcoinClient()}
		info, err := btcClient.GetBlockChainInfo()
		if err != nil {
			configure.Sugar.Fatal("Get info error: ", err.Error())
		}
		configure.Sugar.Info("info: ", info)

		fee, err := btcClient.EstimateFee(int64(6))
		if err != nil {
			configure.Sugar.Warn("EstimateFee: ", err.Error())
		}

		configure.Sugar.Info("fee: ", fee)

		re, err := btcClient.DumpWallet("/root/dumpwallet_private")
		if err != nil {
			configure.Sugar.Warn(err.Error())
		}
		configure.Sugar.Info(re)
		configure.Remote2local("/tmp/btc_wallet_backup", "/root/dumpwallet_private")
	},
}

func main() {
	execute()
}

func init() {
	rootCmd.AddCommand(migrateWallet)
}
