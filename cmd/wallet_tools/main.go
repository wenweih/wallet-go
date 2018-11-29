package main

import (
	"path/filepath"
	"github.com/spf13/cobra"
	"wallet-transition/pkg/configure"
	"wallet-transition/pkg/blockchain"
)

var (
	asset  string
	local  bool
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

var dumpWallet = &cobra.Command{
	Use:   "dump",
	Short: "Migrate wallet from blockchain node",
	Run: func(cmd *cobra.Command, args []string) {
		switch asset {
		case "btc":
			btcClient := blockchain.BitcoinClientAlias{blockchain.NewbitcoinClient()}
			// info, err := btcClient.GetBlockChainInfo()
			// if err != nil {
			// 	configure.Sugar.Fatal("Get info error: ", err.Error())
			// }
			// configure.Sugar.Info("info: ", info)
			//
			// fee, err := btcClient.EstimateFee(int64(6))
			// if err != nil {
			// 	configure.Sugar.Warn("EstimateFee: ", err.Error())
			// }
			//
			// configure.Sugar.Info("fee: ", fee)
			oldWalletServerClient, err := configure.NewServerClient(configure.Config.OldBTCWalletServerUser, configure.Config.OldBTCWalletServerPass, configure.Config.OldBTCWalletServerHost)
			if err != nil {
				configure.Sugar.Fatal(err.Error())
			}

			// create folder for old wallet backup
			if err = oldWalletServerClient.SftpClient.MkdirAll(filepath.Dir(configure.Config.OldBTCWalletFileName)); err != nil {
				configure.Sugar.Fatal(err.Error())
			}

			// dump old wallet to old wallet server
			btcClient.DumpOldWallet(oldWalletServerClient)

			oldWalletServerClient.CopyRemoteFile2(configure.Config.OldBTCWalletFileName, configure.Config.NewBTCWalletFileName, local)

		case "eth":
		default:
			configure.Sugar.Fatal("Only support btc, eth")
			return
		}
	},
}

func main() {
	execute()
}

func init() {
	rootCmd.AddCommand(dumpWallet)
	dumpWallet.Flags().StringVarP(&asset, "asset", "a", "btc", "asset type, support btc, eth")
	dumpWallet.MarkFlagRequired("asset")
	dumpWallet.Flags().BoolVarP(&local, "local", "l", false, "copy dump wallet file to local machine. default copy to remote server, which is set in configure")
}
