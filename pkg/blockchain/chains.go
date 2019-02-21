package blockchain

import (
  "strings"
  "errors"
  "context"
  "wallet-transition/pkg/configure"
  "github.com/ethereum/go-ethereum/common"
  "github.com/ethereum/go-ethereum/accounts/abi/bind"
)

// Balance get specify token balance of an EOA account
func (c EthereumChain) Balance(asset, accountHex string) (string, error) {
  accountAddress := common.HexToAddress(accountHex)
  if strings.ToLower(asset) == configure.ChainsInfo[Ethereum].Coin {
    bal, err := c.Client.BalanceAt(context.Background(), accountAddress, nil)
    if err != nil {
      return "", err
    }
    return bal.String(), nil
  }
  tokenAddress := common.HexToAddress(configure.Config.ETHToken[asset].(string))
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
