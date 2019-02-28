package blockchain

import (
  "strings"
  "wallet-transition/pkg/util"
  "wallet-transition/pkg/configure"
  "github.com/ethereum/go-ethereum/ethclient"
)

// Close close rpc connect
func (client *ETHRPC) Close()  {
  client.Client.Close()
}

// NewEthClient new ethereum rpc client
func NewEthClient() (*ETHRPC, error) {
  ethClient, err := ethclient.Dial(configure.Config.EthRPC)
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

  var ethWalletBackupPath = strings.Join([]string{configure.Config.BackupWalletPath, "eth.backup"}, "")

  if err := oldWalletServerClient.SaveEthAccount(ethWalletBackupPath); err != nil {
    configure.Sugar.Fatal(err.Error())
  }

  oldWalletServerClient.CopyRemoteFile2(ethWalletBackupPath, local)
}
