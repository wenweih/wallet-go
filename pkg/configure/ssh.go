package configure

import (
  "github.com/pkg/sftp"
  "golang.org/x/crypto/ssh"
  "strings"
  "errors"
  "net"
  "io"
  "os"
)

// NewSSHClient connect to ssh server
func NewSSHClient(user, pass, host string)(*ssh.Client, error) {
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

// Remote2 copy file to local or remote server, default copy to server by configure
func Remote2(dstFileWithPath, srcFileWithPath string, toRemote bool)  {
  sshClient, err := NewSSHClient(Config.OldBTCWalletServerUser, Config.OldBTCWalletServerPass, Config.OldBTCWalletServerHost)
  if err != nil {
    Sugar.Fatal(err.Error())
  }
  client, err := sftp.NewClient(sshClient)
  if err != nil {
    Sugar.Fatal("sftp client error: ", err.Error())
  }
  srcFile, err := client.Open(srcFileWithPath)
  if err != nil {
    Sugar.Fatal("Open src file error: ", err.Error())
  }

  dstFile := new(os.File)
  if toRemote{

  }else {
    dstFile, err = os.Create(dstFileWithPath)
    if err != nil {
      Sugar.Fatal("Create dst file error: ", err.Error())
    }
  }

  if _, err := io.Copy(dstFile, srcFile); err != nil {
    Sugar.Fatal("io copy error: ", err.Error())
  }

  if err := dstFile.Sync(); err != nil {
    Sugar.Fatal("dsfFile error: ", err.Error())
  }

}
