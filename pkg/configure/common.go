package configure

import (
  "strings"
  homedir "github.com/mitchellh/go-homedir"
)

// HomeDir 获取服务器当前用户目录路径
func HomeDir() string {
	home, err := homedir.Dir()
	if err != nil {
		Sugar.Fatal(err.Error())
	}
	return home
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
				chaininfo.Coin = strings.ToLower(vv.(string))
				chainAssets[strings.ToLower(vv.(string))] = k
			case "tokens":
				chaininfo.Tokens = make(map[string]string)
				for kt, vt := range vv.(map[string]interface{}) {
					chaininfo.Tokens[kt] = vt.(string)
					chainAssets[strings.ToLower(kt)] = k
				}
      case "accounts":
        chaininfo.Accounts = make(map[string]string)
        for ka, va := range vv.(map[string]interface{}) {
          chaininfo.Accounts[ka] = va.(string)
        }
			}
		}
		chainsInfo[k] = chaininfo
	}
	return chainsInfo, chainAssets
}
