package main

import (
	"io/ioutil"
	"strings"
	"github.com/spf13/cobra"
	"path/filepath"
	"wallet-transition/pkg/blockchain"
	"wallet-transition/pkg/configure"
	"wallet-transition/pkg/db"
	"github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/ethereum/go-ethereum/crypto"
	"encoding/hex"
	"wallet-transition/pkg/util"
)

var (
	asset	string
	local	bool
	pemName	string
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
		util.RsaGen(pemName)
	},
}

var dumpWallet = &cobra.Command {
	Use:   "dump",
	Short: "Dump wallet from blockchain node, upload dump wallet to signed server",
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
			// create folder for old wallet backup
			oldWalletServerClient, err := configure.NewServerClient(configure.Config.OldBTCWalletServerUser, configure.Config.OldBTCWalletServerPass, configure.Config.OldBTCWalletServerHost)
			if err != nil {
				configure.Sugar.Fatal(err.Error())
			}
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

var migrateWallet = &cobra.Command{
	Use:   "migrate",
	Short: "Migrate wallet to levelDB",
	Run: func(cmd *cobra.Command, args []string) {
		switch asset {
		case "btc":
			db.BTCMigrate()
		case "eth":
			oldWalletServerClient, err := configure.NewServerClient(configure.Config.NewETHWalletServerUser, configure.Config.NewETHWalletServerPass, configure.Config.NewETHWalletServerHost)
			if err != nil {
				configure.Sugar.Fatal(err.Error())
			}
			ksFiles, err := oldWalletServerClient.SftpClient.ReadDir(configure.Config.KeystorePath)
			if err != nil {
				configure.Sugar.Fatal("Read keystore directory error: ", configure.Config.KeystorePath, " ", err.Error())
			}

			for _, ks := range ksFiles {
				if strings.HasPrefix(ks.Name(), "UTC"){
					ksFile, err := oldWalletServerClient.SftpClient.Open(strings.Join([]string{configure.Config.KeystorePath, ks.Name()}, "/"))
					if err != nil {
						configure.Sugar.Fatal("Failt to open: ", ks.Name(), " ,", err.Error())
					}
					ksBytes, err := ioutil.ReadAll(ksFile)
					if err != nil {
						configure.Sugar.Fatal("Fail to read ks: ", ks.Name(), ", ", err.Error())
					}
					key, err := keystore.DecryptKey(ksBytes, configure.Config.KSPass)
					if err != nil && strings.Contains(err.Error(), "could not decrypt key with given passphrase"){
						configure.Sugar.Warn("Keystore DecryptKey error: ", err.Error())
					} else {
						priStr := hex.EncodeToString(crypto.FromECDSA(key.PrivateKey))
						configure.Sugar.Info(strings.ToLower(key.Address.String()), " priStr: ", priStr)
					}
					defer ksFile.Close()
				}
			}
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

	migrateWallet.Flags().StringVarP(&asset, "asset", "a", "btc", "asset type, support btc, eth")
	migrateWallet.MarkFlagRequired("asset")

	rsaGenerate.Flags().StringVarP(&pemName, "pem", "p", "", "rsa pem file name")
	rsaGenerate.MarkFlagRequired("pem")
}
