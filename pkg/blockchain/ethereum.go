package blockchain

import (
  "strings"
  "io/ioutil"
  "wallet-transition/pkg/util"
  "wallet-transition/pkg/configure"
)

var ethWalletBackupPath = strings.Join([]string{configure.Config.BackupWalletPath, "eth.backup"}, "")

// DumpETHAccount dump ethereum account from keystore
func DumpETHAccount(local bool)  {
  oldWalletServerClient, err := configure.NewServerClient(configure.Config.OldETHWalletServerUser,
    configure.Config.OldETHWalletServerPass,
    configure.Config.OldETHWalletServerHost)
  if err != nil {
    configure.Sugar.Fatal(err.Error())
  }

  pubBytes, err := ioutil.ReadFile(strings.Join([]string{util.HomeDir(), "dump_wallet_pub.pem"}, "/"))
  if err != nil {
    configure.Sugar.Fatal(err.Error())
  }
  rsaPub := util.BytesToPublicKey(pubBytes)

  if err := oldWalletServerClient.SaveEncryptedEthAccount(rsaPub); err != nil {
    configure.Sugar.Fatal(err.Error())
  }
}
