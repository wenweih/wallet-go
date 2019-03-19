package configure

import (
	"time"
	"github.com/spf13/viper"
	"go.uber.org/zap"
)

const (
	// LDBPath leveldb folder name for key-pair
	LDBPath = ".db_wallet"
	// BackUpWalletPath backup wallet path
	BackUpWalletPath = "/usr/local/wallet-go/"
	// ConfigureFile configure file name
	ConfigureFile = "wallet-go"
)

// InitConfig 配置信息
func InitConfig() *Configure {
	var conf Configure
	viper.SetConfigType("yaml")
	viper.AddConfigPath(HomeDir())
	viper.SetConfigName(ConfigureFile)
	viper.AutomaticEnv() // read in environment variables that match

	conf.DBWalletPath = LDBPath
	conf.BackupWalletPath = BackUpWalletPath

	// If a config file is found, read it in.
	err := viper.ReadInConfig()
	if err == nil {
		Sugar.Info("Using Configure file: ", viper.ConfigFileUsed(), " Time: ", time.Now().Format("Mon Jan _2 15:04:05 2006"))
	} else {
		Sugar.Fatal("Error: wallet-go not found in: ", HomeDir(), err.Error())
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

		case "omni_node_host":
			conf.OmniNODEHOST = value.(string)
		case "omni_node_usr":
			conf.OmniNODEUSR = value.(string)
		case "omni_node_pass":
			conf.OmniNODEPASS = value.(string)
		case "omni_http_mode":
			conf.OmniHTTPPostMode = value.(bool)
		case "omni_disable_tls":
			conf.OmniDisableTLS = value.(bool)

		case "eth_rpc_ws":
			conf.EthRPCWS = value.(string)
		case "eth_rpc":
			conf.EthRPC = value.(string)

		case "eosio_rpc":
			conf.EOSIORPC = value.(string)

		case "db_mysql_host":
			conf.MySQLHost = value.(string)
		case "db_mysql_user":
			conf.MySQLUser = value.(string)
		case "db_mysql_pass":
			conf.MySQLPass = value.(string)
		case "db_mysql_name":
			conf.MySQLName = value.(string)

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
		case "wallet_core_rpc_url":
			conf.WalletCoreRPCURL = value.(string)
		case "chains":
			conf.Chains = viper.Sub("chains").AllSettings()
		}
	}
	return &conf
}

func init() {
	Config = new(Configure)
	Sugar = zap.NewExample().Sugar()
	defer Sugar.Sync()
	Config = InitConfig()
	ChainsInfo, ChainAssets = Config.ChainConfigInfo()
}
