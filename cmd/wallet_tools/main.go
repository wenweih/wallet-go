package main

import (
	"github.com/spf13/cobra"
	"wallet-go/pkg/blockchain"
	"wallet-go/pkg/configure"
	"wallet-go/pkg/db"
	"wallet-go/pkg/util"
)

var (
	asset	string
	local	bool
	utxo bool
)

var rootCmd = &cobra.Command {
	Use:   "wallet-go-tool",
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
		util.RsaGen("wallet")
		configure.Sugar.Info("Generate rsa pub/priv pem successfully")
	},
}

var dumpWallet = &cobra.Command {
	Use:   "dump",
	Short: "Dump wallet from blockchain node, upload dump wallet to signed server",
	Run: func(cmd *cobra.Command, args []string) {
		switch asset {
		case "btc":
			client, err := blockchain.NewbitcoinClient()
			if err != nil {
				configure.Sugar.Fatal(err.Error())
			}
			btcClient := blockchain.BTCRPC{Client: client}
			if utxo {
				btcClient.DumpUTXO()
				return
			}
			btcClient.DumpBTC(local)
		case "eth":
			blockchain.DumpETHAccount(local)
		default:
			configure.Sugar.Fatal("Only support btc, eth")
			return
		}
	},
}

var migrateWallet = &cobra.Command {
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
			ldb, err := db.NewLDB("eth")
			if err != nil {
				configure.Sugar.Fatal(err.Error())
			}
			ldb.MigrateETH()
			ldb.Close()
		default:
			configure.Sugar.Fatal("Only support btc, eth")
			return
		}
	},
}

var importPrivateKey = &cobra.Command {
	Use:   "import",
	Short: "import private key",
	Run: func(cmd *cobra.Command, args []string){
		client, err := blockchain.NewbitcoinClient()
		if err != nil {
			configure.Sugar.Fatal(err.Error())
		}
		btcClient := blockchain.BTCRPC{Client: client}
		btcClient.ImportPrivateKey()
	},
}

func main() {
	execute()
}

func init() {
	rootCmd.AddCommand(dumpWallet, migrateWallet, rsaGenerate, importPrivateKey)
	dumpWallet.Flags().StringVarP(&asset, "asset", "a", "btc", "asset type, support btc, eth")
	dumpWallet.MarkFlagRequired("asset")
	dumpWallet.Flags().BoolVarP(&local, "local", "l", false, "copy dump wallet file to local machine. default copy to remote server, which is set in configure")
	dumpWallet.Flags().BoolVarP(&utxo, "utxo", "u", false, "Dump utxo")

	migrateWallet.Flags().StringVarP(&asset, "asset", "a", "", "asset type, support btc, eth")
	migrateWallet.MarkFlagRequired("asset")
}
