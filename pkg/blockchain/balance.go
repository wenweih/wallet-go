package blockchain

import (
  "fmt"
  "strings"
  "context"
  "strconv"
  "encoding/json"
  "github.com/btcsuite/btcutil"
  "wallet-go/pkg/configure"
  "github.com/ethereum/go-ethereum/common"
  "github.com/ethereum/go-ethereum/accounts/abi/bind"
)

// Balance omnicore protocol token balance query
func (c BitcoinCoreChain) Balance(ctx context.Context, account, symbol, code string) (string, error) {
  token := configure.ChainsInfo[Bitcoin].Tokens[symbol]
  propertyid, err := strconv.Atoi(token)
  if err != nil {
    return "", fmt.Errorf("Convert to propertyid %s", err)
  }

  _, err = btcutil.DecodeAddress(account, c.Mode)
  if err != nil {
    return "", fmt.Errorf("Illegal Bitcoin Address %s : %s", account, err)
  }

  var params []json.RawMessage
  {
    address, err := json.Marshal(account)
    if err != nil {
      return "", err
    }
    perpertyID, err := json.Marshal(propertyid)
    if err != nil {
      return "", err
    }
    params = []json.RawMessage{address, perpertyID}
  }

  info, err := c.Client.RawRequest("omni_getbalance", params)
	if err != nil {
		return "", err
	}

	var omniBalance OmniBalance
	if err := json.Unmarshal(info, &omniBalance); err != nil {
		return "", err
	}
  return omniBalance.Balance, nil
}

// Balance get specify token balance of an Ethereum EOA account
func (c EthereumChain) Balance(ctx context.Context, account, symbol, code string) (string, error) {
  accountAddress := common.HexToAddress(account)
  if strings.ToLower(symbol) == strings.ToLower(configure.ChainsInfo[Ethereum].Coin) {
    bal, err := c.Client.BalanceAt(context.Background(), accountAddress, nil)
    if err != nil {
      return "", err
    }
    return bal.String(), nil
  }
  token := configure.ChainsInfo[Ethereum].Tokens[strings.ToLower(symbol)]
  if token == "" {
    return "", fmt.Errorf("Token not implement yet: %s", symbol)
  }
  tokenAddress := common.HexToAddress(token)
  contractInstance, err := NewEthToken(tokenAddress, c.Client)
  if err != nil {
    return "", fmt.Errorf("Get token instance %s", err)
  }
  bal, err := contractInstance.BalanceOf(&bind.CallOpts{}, accountAddress)
  if err!= nil {
    return "", fmt.Errorf("Get token balance %s", err)
  }
  return bal.String(), nil
}

// Balance EOSIO balance query
func (c EOSChain) Balance(ctx context.Context, account, symbol, code string) (string, error) {
  accountName, err := ToAccountNameEOS(account)
  if err != nil {
    return "", err
  }
  codeName, err := ToAccountNameEOS(code)
  if err != nil {
    return "", err
  }
  balances, err := c.Client.GetCurrencyBalance(accountName, symbol, codeName)
  if err != nil {
    return "", err
  }
  if len(balances) > 0 {
    return balances[0].String(), nil
  }
  return "", fmt.Errorf("Balance not found")
}
