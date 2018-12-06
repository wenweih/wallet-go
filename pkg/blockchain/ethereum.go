package blockchain

import (
  "strings"
  "io/ioutil"
  "encoding/hex"
  "wallet-transition/pkg/util"
  "wallet-transition/pkg/configure"
  "github.com/ethereum/go-ethereum/crypto"
  "github.com/ethereum/go-ethereum/accounts/keystore"
)

// DumpETHAccount dump ethereum account from keystore
func DumpETHAccount(local bool)  {
  oldWalletServerClient, err := configure.NewServerClient(configure.Config.OldETHWalletServerUser,
    configure.Config.OldETHWalletServerPass,
    configure.Config.OldETHWalletServerHost)
  if err != nil {
    configure.Sugar.Fatal(err.Error())
  }
  ksFiles, err := oldWalletServerClient.SftpClient.ReadDir(configure.Config.KeystorePath)
  if err != nil {
    configure.Sugar.Fatal("Read keystore directory error: ", configure.Config.KeystorePath, " ", err.Error())
  }

  for _, ks := range ksFiles {
    if strings.HasPrefix(ks.Name(), "UTC"){
      ksFile, err := oldWalletServerClient.SftpClient.Open(strings.Join([]string{configure.Config.KeystorePath, ks.Name()}, "/"))
      if err != nil {
        configure.Sugar.Fatal("Failt to open: ", ks.Name(), " ,", err.Error())
      }
      ksBytes, err := ioutil.ReadAll(ksFile)
      if err != nil {
        configure.Sugar.Fatal("Fail to read ks: ", ks.Name(), ", ", err.Error())
      }
      key, err := keystore.DecryptKey(ksBytes, configure.Config.KSPass)
      if err != nil && strings.Contains(err.Error(), "could not decrypt key with given passphrase"){
        configure.Sugar.Warn("Keystore DecryptKey error: ", err.Error())
      } else {
        pubBytes, err := ioutil.ReadFile(strings.Join([]string{util.HomeDir(), "dump_wallet_pub.pem"}, "/"))
        if err != nil {
          configure.Sugar.Fatal(err.Error())
        }
        rsaPub := util.BytesToPublicKey(pubBytes)
        encryptAccountPriv := util.EncryptWithPublicKey(crypto.FromECDSA(key.PrivateKey), rsaPub)
        configure.Sugar.Info(hex.EncodeToString(encryptAccountPriv))
      }
      defer ksFile.Close()
    }
  }
}
