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
	EthRPCWS        string
	EthRPCHTTP      string
	BTCRPCHTTP      string
	BTCUser         string
	BTCPass         string
	BTCHTTPPostMode bool
	BTCDisableTLS   bool
	MQ              string
	DB              string
	WalletRPCURL    string
	ElasticURL      string
	ElasticSniff    bool
	ETHTXINDEX      string
	Redis           string
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
		case "eth_rpc_ws":
			conf.EthRPCWS = value.(string)
		case "eth_rpc_http":
			conf.EthRPCHTTP = value.(string)
		case "mq":
			conf.MQ = value.(string)
		case "wallet_rpc_url":
			conf.WalletRPCURL = value.(string)
		case "elastic_url":
			conf.ElasticURL = value.(string)
		case "elastic_sniff":
			conf.ElasticSniff = value.(bool)
		case "db_mysql":
			conf.DB = value.(string)
		case "eth_tx_index":
			conf.ETHTXINDEX = value.(string)
		case "btc_host":
			conf.BTCRPCHTTP = value.(string)
		case "btc_usr":
			conf.BTCUser = value.(string)
		case "btc_pass":
			conf.BTCPass = value.(string)
		case "btc_http_mode":
			conf.BTCHTTPPostMode = value.(bool)
		case "btc_disable_tls":
			conf.BTCDisableTLS = value.(bool)
		case "redis":
			conf.Redis = value.(string)
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
