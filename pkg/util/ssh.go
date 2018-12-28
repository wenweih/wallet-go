package util

import (
	"io"
	"os"
	"net"
	"strings"
	"errors"
	"syscall"
	"io/ioutil"
	"encoding/hex"
	"path/filepath"
	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
	"wallet-transition/pkg/configure"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/accounts/keystore"
)

// ServerClient includs ssh client and sftp client
type ServerClient struct {
	SSHClient  *ssh.Client
	SftpClient *sftp.Client
}

// Close close ssh and sftp conn
func (c *ServerClient) Close() error {
	if err := c.SftpClient.Close(); err != nil {
		return errors.New("ServerClient close sftp client error")
	}

	if err := c.SSHClient.Close(); err != nil {
		return errors.New("ServerClient close ssh client error")
	}
	return nil
}

// NewServerClient server client object
func NewServerClient(user, pass, host string) (*ServerClient, error) {
	sshClient, err := newSSHClient(user, pass, host)
	if err != nil {
		return nil, errors.New(strings.Join([]string{"sshClient error: ", err.Error()}, ""))
	}
	sftpClient, err := sftp.NewClient(sshClient)
	if err != nil {
		return nil, errors.New(strings.Join([]string{"sftpClient error: ", err.Error()}, ""))
	}
	return &ServerClient{sshClient, sftpClient}, nil
}

// connect to ssh server
func newSSHClient(user, pass, host string) (*ssh.Client, error) {
	sshConfig := &ssh.ClientConfig{
		User: user,
		Auth: []ssh.AuthMethod{ssh.Password(pass)},
		HostKeyCallback: func(hostname string, remote net.Addr, key ssh.PublicKey) error {
			return nil
		},
	}
	sshConfig.HostKeyCallback = ssh.InsecureIgnoreHostKey()
	client, err := ssh.Dial("tcp", host, sshConfig)
	if err != nil {
		return nil, errors.New(strings.Join([]string{"ssh dial error: ", err.Error()}, ""))
	}
	return client, nil
}

// CopyRemoteFile2 copy file to local or remote server, default copy to server by configure
func (c *ServerClient) CopyRemoteFile2(backupPath string, local bool) {
	// http://networkbit.ch/golang-sftp-client/
	srcFile, err := c.SftpClient.Open(backupPath)
	if err != nil {
		configure.Sugar.Fatal("Open src file error: ", err.Error())
	}
	if local {
		defer syscall.Umask(syscall.Umask(0))
		if err := os.MkdirAll(filepath.Dir(backupPath), 0700); err != nil {
			configure.Sugar.Fatal("MkdirAll error: ", err.Error())
		}
		path := strings.Join([]string{backupPath, "new"}, "_")
		dstFile, err := os.Create(path)
		if err != nil {
			configure.Sugar.Fatal("Create dst file error: ", err.Error())
		}

		if _, err := io.Copy(dstFile, srcFile); err != nil {
			configure.Sugar.Fatal("io copy error: ", err.Error())
		}
		if err := dstFile.Sync(); err != nil {
			configure.Sugar.Fatal("dsfFile error: ", err.Error())
		}
		defer dstFile.Close()
		configure.Sugar.Info("Copy to local: ", path)
	} else {
		newWalletServerClient, err := NewServerClient(configure.Config.NewWalletServerUser, configure.Config.NewWalletServerPass, configure.Config.NewWalletServerHost)
		if err != nil {
			configure.Sugar.Fatal(err.Error())
		}

		// create folder for old wallet backup in new server
		if err = newWalletServerClient.SftpClient.MkdirAll(filepath.Dir(backupPath)); err != nil {
			configure.Sugar.Fatal(err.Error())
		}
		dstFile, err := newWalletServerClient.SftpClient.Create(strings.Join([]string{backupPath, "new"}, "_"))
		defer dstFile.Close()

		if _, err := io.Copy(dstFile, srcFile); err != nil {
			configure.Sugar.Fatal("Copy dst file error: ", err.Error())
		}

		if err := newWalletServerClient.Close(); err != nil {
			configure.Sugar.Fatal(err.Error())
		}
		configure.Sugar.Info("Copy to new server: ", newWalletServerClient.SSHClient.RemoteAddr().String(), ":", strings.Join([]string{backupPath, "new"}, "_"))
	}

	defer srcFile.Close()
	if err := c.Close(); err != nil {
		configure.Sugar.Fatal(err.Error())
	}
}

// SaveEncryptedEthAccount save ethereum account to file
func (c *ServerClient) SaveEthAccount(ethWalletBackupPath string)  error {
	// create folder for old wallet backup
	if err := c.SftpClient.MkdirAll(filepath.Dir(ethWalletBackupPath)); err != nil {
		return errors.New(strings.Join([]string{"Create", ethWalletBackupPath , "directory error", err.Error()}, " "))
	}

	srcBackupFile, err := c.SftpClient.OpenFile(ethWalletBackupPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY)
	if err != nil {
		return errors.New(strings.Join([]string{"open remote eth.backup error", err.Error()}, " "))
	}
	defer srcBackupFile.Close()

	ksFiles, err := c.SftpClient.ReadDir(configure.Config.KeystorePath)
	if err != nil {
		return errors.New(strings.Join([]string{"Read keystore directory error", configure.Config.KeystorePath, err.Error()}, " "))
	}

  for _, ks := range ksFiles {
    if strings.HasPrefix(ks.Name(), "UTC"){
      ksFile, err := c.SftpClient.Open(strings.Join([]string{configure.Config.KeystorePath, ks.Name()}, "/"))
      if err != nil {
				return errors.New(strings.Join([]string{"Failt to open", ks.Name(), err.Error()}, " "))
      }
      ksBytes, err := ioutil.ReadAll(ksFile)
      if err != nil {
				return errors.New(strings.Join([]string{"Fail to read ks", ks.Name(), err.Error()}, " "))
      }
      key, err := keystore.DecryptKey(ksBytes, configure.Config.KSPass)
      if err != nil && strings.Contains(err.Error(), "could not decrypt key with given passphrase"){
        configure.Sugar.Warn("Keystore DecryptKey error: ", err.Error())
      } else {
        address := key.Address.String()
        // encryptAccountPriv := EncryptWithPublicKey(crypto.FromECDSA(key.PrivateKey), rsaPub)
        fileData := []byte(strings.Join([]string{address, hex.EncodeToString(crypto.FromECDSA(key.PrivateKey))}, " "))
        fileData = append(fileData, '\n')
        n, err := srcBackupFile.Write(fileData)
        if err != nil {
					return errors.New(strings.Join([]string{"write eth backup file error", err.Error()}, " "))
        }
        if err == nil && n < len(fileData) {
          err = io.ErrShortWrite
        }
        configure.Sugar.Info("Ethereum account: ", address)
      }
      defer ksFile.Close()
    }
  }
	return nil
}
