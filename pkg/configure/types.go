package configure

import (
	"go.uber.org/zap"
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

// Configure 配置数据
type Configure struct {
	BTCNODEHOST     string
	BTCNODEUSR      string
	BTCNODEPASS     string

	OmniNODEHOST     string
	OmniNODEUSR      string
	OmniNODEPASS     string
	OmniHTTPPostMode bool
	OmniDisableTLS   bool

	EOSIORPC         string

	EthRPCWS        string
	EthRPC          string

	MySQLHost       string
	MySQLUser       string
	MySQLPass       string
	MySQLName       string

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

	WalletCoreRPCURL        string

	Chains                  map[string]interface{}

	MQ                       string
}

// ChainInfo chain info
type ChainInfo struct {
	Confirmations int
	Chain         string
	Coin          string
	Tokens        map[string]string
	Accounts      map[string]string
}
