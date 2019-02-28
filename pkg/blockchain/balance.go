package blockchain

import (
  "fmt"
  "strings"
  "errors"
  "context"
  "wallet-transition/pkg/configure"
  "github.com/ethereum/go-ethereum/common"
  "github.com/ethereum/go-ethereum/accounts/abi/bind"
)

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
    return "", errors.New(strings.Join([]string{"Get token instance error: ", err.Error()}, ""))
  }
  bal, err := contractInstance.BalanceOf(&bind.CallOpts{}, accountAddress)
  if err!= nil {
    return "", errors.New(strings.Join([]string{"Get token balance error: ", err.Error()}, ""))
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
  return "", errors.New("balance not found")
}
