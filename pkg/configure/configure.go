package configure

import (
	"time"
	"github.com/spf13/viper"
	"go.uber.org/zap"
	homedir "github.com/mitchellh/go-homedir"
)

var (
	// Sugar log
	Sugar *zap.SugaredLogger
	// Config configure
	Config *Configure
)

// HomeDir 获取服务器当前用户目录路径
func HomeDir() string {
	home, err := homedir.Dir()
	if err != nil {
		Sugar.Fatal(err.Error())
	}
	return home
}

// Configure 配置数据
type Configure struct {
	BTCNODEHOST     string
	BTCNODEUSR      string
	BTCNODEPASS     string
	BTCHTTPPostMode bool
	BTCDisableTLS   bool

	EthRPCWS        string

	OldBTCWalletServerHost	string
	OldBTCWalletServerUser	string
	OldBTCWalletServerPass	string

	DBWalletPath 			  		string
	BackupWalletPath				string

	NewWalletServerHost			string
	NewWalletServerUser			string
	NewWalletServerPass			string

	OldETHWalletServerHost	string
	OldETHWalletServerUser	string
	OldETHWalletServerPass	string
	KeystorePath            string
	KSPass                  string

	APIASSETS               []string
	WalletCoreRPCURL        string

	ETHToken                map[string]interface{}
}

// InitConfig 配置信息
func InitConfig() *Configure {
	var conf Configure
	viper.SetConfigType("yaml")
	viper.AddConfigPath(HomeDir())
	viper.SetConfigName("wallet-transition")
	viper.AutomaticEnv() // read in environment variables that match

	conf.DBWalletPath = ".db_wallet"
	conf.BackupWalletPath = "/usr/local/wallet-transition/"

	// If a config file is found, read it in.
	err := viper.ReadInConfig()
	if err == nil {
		Sugar.Info("Using Configure file: ", viper.ConfigFileUsed(), " Time: ", time.Now().Format("Mon Jan _2 15:04:05 2006"))
	} else {
		Sugar.Fatal("Error: wallet-service not found in: ", HomeDir())
	}

	for key, value := range viper.AllSettings() {
		switch key {
		// BTC node info
		case "btc_node_host":
			conf.BTCNODEHOST = value.(string)
		case "btc_node_usr":
			conf.BTCNODEUSR = value.(string)
		case "btc_node_pass":
			conf.BTCNODEPASS = value.(string)
		case "btc_http_mode":
			conf.BTCHTTPPostMode = value.(bool)
		case "btc_disable_tls":
			conf.BTCDisableTLS = value.(bool)

		case "eth_rpc_ws":
			conf.EthRPCWS = value.(string)

		// old btc wallet server info
		case "old_btc_wallet_server_host":
			conf.OldBTCWalletServerHost = value.(string)
		case "old_btc_wallet_server_user":
			conf.OldBTCWalletServerUser = value.(string)
		case "old_btc_wallet_server_pass":
			conf.OldBTCWalletServerPass = value.(string)

		// new btc wallet server info
		case "new_wallet_server_host":
			conf.NewWalletServerHost = value.(string)
		case "new_wallet_server_user":
			conf.NewWalletServerUser = value.(string)
		case "new_wallet_server_pass":
			conf.NewWalletServerPass = value.(string)

		// ethereum wallet
		case "old_eth_wallet_server_host":
			conf.OldETHWalletServerHost = value.(string)
		case "old_eth_wallet_server_user":
			conf.OldETHWalletServerUser = value.(string)
		case "old_eth_wallet_server_pass":
			conf.OldETHWalletServerPass = value.(string)
		case "keystore_path":
			conf.KeystorePath = value.(string)
		case "ks_pass":
			conf.KSPass = value.(string)

		case "api_assets":
			conf.APIASSETS = viper.GetStringSlice(key)

		case "wallet_core_rpc_url":
			conf.WalletCoreRPCURL = value.(string)

		case "eth_token":
			conf.ETHToken = viper.Sub("eth_token").AllSettings()
		}
	}
	return &conf
}

func init() {
	Config = new(Configure)
	Sugar = zap.NewExample().Sugar()
	defer Sugar.Sync()
	Config = InitConfig()
}
