package configure

import (
	"time"
	"strings"
	"github.com/spf13/viper"
	"go.uber.org/zap"
	homedir "github.com/mitchellh/go-homedir"
)

var (
	// Sugar log
	Sugar *zap.SugaredLogger
	// Config configure
	Config *Configure
	// ChainsInfo chain info
	ChainsInfo map[string]ChainInfo
	// ChainAssets chain assets
	ChainAssets map[string]string
)

// HomeDir 获取服务器当前用户目录路径
func HomeDir() string {
	home, err := homedir.Dir()
	if err != nil {
		Sugar.Fatal(err.Error())
	}
	return home
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
		Sugar.Fatal("Error: wallet-transition not found in: ", HomeDir(), err.Error())
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

		case "api_assets":
			conf.APIASSETS = viper.GetStringSlice(key)

		case "wallet_core_rpc_url":
			conf.WalletCoreRPCURL = value.(string)

		case "eth_token":
			conf.ETHToken = viper.Sub("eth_token").AllSettings()

		case "omni_token":
			conf.OmniToken = viper.Sub("omni_token").AllSettings()

		case "chains":
			conf.Chains = viper.Sub("chains").AllSettings()

		case "confirmations":
			conf.Confirmations = viper.Sub("confirmations").AllSettings()
		}
	}
	return &conf
}

// ChainConfigInfo chain info
func (c *Configure) ChainConfigInfo() (map[string]ChainInfo, map[string]string) {
	chains := c.Chains
	var (
		chainsInfo = make(map[string]ChainInfo)
		chainAssets = make(map[string]string)
	)

	for k, v := range chains {
		var chaininfo  ChainInfo
		for kv, vv := range v.(map[string]interface{}) {
			switch kv {
			case "confirmations":
				chaininfo.Confirmations = vv.(int)
			case "coin":
				chaininfo.Coin = vv.(string)
				chainAssets[strings.ToLower(vv.(string))] = k
			case "tokens":
				chaininfo.Tokens = make(map[string]string)
				for kt, vt := range vv.(map[string]interface{}) {
					chaininfo.Tokens[kt] = vt.(string)
					chainAssets[strings.ToLower(kt)] = k
				}
			}
		}
		chainsInfo[k] = chaininfo
	}
	return chainsInfo, chainAssets
}

func init() {
	Config = new(Configure)
	Sugar = zap.NewExample().Sugar()
	defer Sugar.Sync()
	Config = InitConfig()
	ChainsInfo, ChainAssets = Config.ChainConfigInfo()
}
