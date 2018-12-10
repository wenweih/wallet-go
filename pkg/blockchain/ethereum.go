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
