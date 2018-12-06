package main

import (
	"github.com/spf13/cobra"
	"wallet-transition/pkg/blockchain"
	"wallet-transition/pkg/configure"
	"wallet-transition/pkg/db"
	"wallet-transition/pkg/util"
)

var (
	asset	string
	local	bool

)

var rootCmd = &cobra.Command {
	Use:   "wallet-transition-tool",
	Short: "Commandline to for anbi exchange wallet module",
}

func execute() {
	if err := rootCmd.Execute(); err != nil {
		configure.Sugar.Fatal("Command execute error: ", err.Error())
	}
}

var rsaGenerate = &cobra.Command {
	Use:   "grsa",
	Short: "Generate rsa key, save to current user home path",
	Run: func(cmd *cobra.Command, args []string) {
		util.RsaGen("dump_wallet")
		configure.Sugar.Info("Generate rsa pub/priv pem successfully")
	},
}

var dumpWallet = &cobra.Command {
	Use:   "dump",
	Short: "Dump wallet from blockchain node, upload dump wallet to signed server",
	Run: func(cmd *cobra.Command, args []string) {
		switch asset {
		case "btc":
			btcClient := blockchain.BTCRPC{Client: blockchain.NewbitcoinClient()}
			btcClient.DumpBTC(local)
		case "eth":
			blockchain.DumpETHAccount(false)
		default:
			configure.Sugar.Fatal("Only support btc, eth")
			return
		}
	},
}

var migrateWallet = &cobra.Command{
	Use:   "migrate",
	Short: "Migrate wallet to levelDB",
	Run: func(cmd *cobra.Command, args []string) {
		switch asset {
		case "btc":
			ldb, err := db.NewLDB("btc")
			if err != nil {
				configure.Sugar.Fatal(err.Error())
			}
			ldb.MigrateBTC()
			ldb.Close()
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
	rootCmd.AddCommand(dumpWallet, migrateWallet, rsaGenerate)
	dumpWallet.Flags().StringVarP(&asset, "asset", "a", "btc", "asset type, support btc, eth")
	dumpWallet.MarkFlagRequired("asset")
	dumpWallet.Flags().BoolVarP(&local, "local", "l", false, "copy dump wallet file to local machine. default copy to remote server, which is set in configure")

	migrateWallet.Flags().StringVarP(&asset, "asset", "a", "", "asset type, support btc, eth")
	migrateWallet.MarkFlagRequired("asset")
}
