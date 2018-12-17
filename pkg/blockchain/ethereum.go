package blockchain

import (
  "errors"
  "strings"
  "io/ioutil"
  "context"
  "math/big"
  "wallet-transition/pkg/util"
  "wallet-transition/pkg/configure"
  "github.com/ethereum/go-ethereum/common"
  "github.com/ethereum/go-ethereum/ethclient"
)

// ETHRPC bitcoin-core client alias
type ETHRPC struct {
	Client *ethclient.Client
}

// Close close rpc connect
func (client *ETHRPC) Close()  {
  client.Client.Close()
}

// NewEthClient new ethereum rpc client
func NewEthClient() (*ETHRPC, error) {
  ethClient, err := ethclient.Dial(configure.Config.EthRPCWS)
  if err != nil {
    return nil, err
  }
  return &ETHRPC{ethClient}, nil
}

var ethWalletBackupPath = strings.Join([]string{configure.Config.BackupWalletPath, "eth.backup"}, "")

// DumpETHAccount dump ethereum account from keystore
func DumpETHAccount(local bool)  {
  oldWalletServerClient, err := util.NewServerClient(configure.Config.OldETHWalletServerUser,
    configure.Config.OldETHWalletServerPass,
    configure.Config.OldETHWalletServerHost)
  if err != nil {
    configure.Sugar.Fatal(err.Error())
  }

  pubBytes, err := ioutil.ReadFile(strings.Join([]string{configure.HomeDir(), "wallet_pub.pem"}, "/"))
  if err != nil {
    configure.Sugar.Fatal(err.Error())
  }
  rsaPub := util.BytesToPublicKey(pubBytes)

  var ethWalletBackupPath = strings.Join([]string{configure.Config.BackupWalletPath, "eth.backup"}, "")

  if err := oldWalletServerClient.SaveEncryptedEthAccount(ethWalletBackupPath, rsaPub); err != nil {
    configure.Sugar.Fatal(err.Error())
  }

  oldWalletServerClient.CopyRemoteFile2(ethWalletBackupPath, local)
}

// GetBalanceAndPendingNonceAtAndGasPrice construct eth tx info
func (client *ETHRPC) GetBalanceAndPendingNonceAtAndGasPrice(ctx context.Context, address string) (*big.Int, *uint64, *big.Int, error) {
	balance, err := client.Client.BalanceAt(ctx, common.HexToAddress(address), nil)
	if err != nil {
		return nil, nil, nil, errors.New(strings.Join([]string{"Failed to get ethereum balance from address:", address, err.Error()}, " "))
	}

	pendingNonceAt, err := client.Client.PendingNonceAt(ctx, common.HexToAddress(address))
	if err != nil {
		return nil, nil, nil, errors.New(strings.Join([]string{"Failed to get account nonce from address:", address, err.Error()}, " "))
	}

	gasPrice, err := client.Client.SuggestGasPrice(ctx)
	if err != nil {
		return nil, nil, nil, errors.New(strings.Join([]string{"get gasPrice error", err.Error()}, " "))
	}

	return balance, &pendingNonceAt, gasPrice, nil

}
