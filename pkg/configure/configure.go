package configure

import (
	"time"
	"wallet-transition/pkg/util"

	"github.com/spf13/viper"
	"go.uber.org/zap"
)

var (
	// Sugar log
	Sugar *zap.SugaredLogger
	// Config configure
	Config *Configure
)

// Configure 配置数据
type Configure struct {
	BTCNODEHOST             string
	BTCNODEUSR              string
	BTCNODEPASS      				string
	BTCHTTPPostMode 				bool
	BTCDisableTLS   				bool

	OldBTCWalletServerHost	string
	OldBTCWalletServerUser	string
	OldBTCWalletServerPass  string
	OldBTCWalletFileName		string

	NewBTCWalletServerHost	string
	NewBTCWalletServerUser	string
	NewBTCWalletServerPass  string
	NewBTCWalletFileName		string

	DBBTCWalletPath         string
}

// InitConfig 配置信息
func InitConfig() *Configure {
	var conf Configure
	viper.SetConfigType("yaml")
	viper.AddConfigPath(util.HomeDir())
	viper.SetConfigName("wallet-transition")
	viper.AutomaticEnv() // read in environment variables that match

	// If a config file is found, read it in.
	err := viper.ReadInConfig()
	if err == nil {
		Sugar.Info("Using Configure file: ", viper.ConfigFileUsed(), " Time: ", time.Now().Format("Mon Jan _2 15:04:05 2006"))
	} else {
		Sugar.Fatal("Error: wallet-service not found in: ", util.HomeDir())
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

		// old btc wallet server info
		case "old_btc_wallet_server_host":
			conf.OldBTCWalletServerHost = value.(string)
		case "old_btc_wallet_server_user":
			conf.OldBTCWalletServerUser = value.(string)
		case "old_btc_wallet_server_pass":
			conf.OldBTCWalletServerPass = value.(string)
		case "old_btc_wallet_file_name":
			conf.OldBTCWalletFileName = value.(string)

		// new btc wallet server info
		case "new_btc_wallet_server_host":
			conf.NewBTCWalletServerHost = value.(string)
		case "new_btc_wallet_server_user":
			conf.NewBTCWalletServerUser = value.(string)
		case "new_btc_wallet_server_pass":
			conf.NewBTCWalletServerPass = value.(string)
		case "new_btc_wallet_file_name":
			conf.NewBTCWalletFileName = value.(string)

		// DB
		case "db_btc_wallet_path":
			conf.DBBTCWalletPath = value.(string)
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
