package configure


// Configure 配置数据
type Configure struct {
	BTCNODEHOST     string
	BTCNODEUSR      string
	BTCNODEPASS     string
	BTCHTTPPostMode bool
	BTCDisableTLS   bool

	OmniNODEHOST     string
	OmniNODEUSR      string
	OmniNODEPASS     string
	OmniHTTPPostMode bool
	OmniDisableTLS   bool

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

	APIASSETS               []string
	WalletCoreRPCURL        string

	ETHToken                map[string]interface{}
	OmniToken               map[string]interface{}

	Chains                  map[string]interface{}

	Confirmations           map[string]interface{}
}

// ChainInfo chain info
type ChainInfo struct {
	Confirmations int
	Chain         string
	Coin          string
	Tokens        map[string]string
}
