package blockchain

import (
  "io"
  "os"
  "strings"
  "io/ioutil"
  "encoding/hex"
  "path/filepath"
  "wallet-transition/pkg/util"
  "wallet-transition/pkg/configure"
  "github.com/ethereum/go-ethereum/crypto"
  "github.com/ethereum/go-ethereum/accounts/keystore"
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
  ksFiles, err := oldWalletServerClient.SftpClient.ReadDir(configure.Config.KeystorePath)
  if err != nil {
    configure.Sugar.Fatal("Read keystore directory error: ", configure.Config.KeystorePath, " ", err.Error())
  }

  // create folder for old wallet backup
  if err = oldWalletServerClient.SftpClient.MkdirAll(filepath.Dir(ethWalletBackupPath)); err != nil {
    configure.Sugar.Fatal(err.Error())
  }

  srcBackupFile, err := oldWalletServerClient.SftpClient.OpenFile(ethWalletBackupPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY)
  if err != nil {
    configure.Sugar.Fatal("open remote eth.backup error", err.Error())
  }
  defer srcBackupFile.Close()

  pubBytes, err := ioutil.ReadFile(strings.Join([]string{util.HomeDir(), "dump_wallet_pub.pem"}, "/"))
  if err != nil {
    configure.Sugar.Fatal(err.Error())
  }
  rsaPub := util.BytesToPublicKey(pubBytes)

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
        address := key.Address.String()
        encryptAccountPriv := util.EncryptWithPublicKey(crypto.FromECDSA(key.PrivateKey), rsaPub)
        fileData := []byte(strings.Join([]string{address, hex.EncodeToString(encryptAccountPriv)}, " "))
        fileData = append(fileData, '\n')
        n, err := srcBackupFile.Write(fileData)
        if err != nil {
          configure.Sugar.Fatal("write eth backup file error: ", err.Error())
        }
        if err == nil && n < len(fileData) {
          err = io.ErrShortWrite
        }
        configure.Sugar.Info("Ethereum account: ", address)
      }
      defer ksFile.Close()
    }
  }
}
