package configure

import (
  "github.com/pkg/sftp"
  "golang.org/x/crypto/ssh"
  "strings"
  "net/url"
  "errors"
  "net"
  "io"
	// "io/ioutil"
  // "bytes"
  // "path"
  "os"
)

// NewSSHClient connect to ssh server
func NewSSHClient()(*ssh.Client, error) {
	sshConfig := &ssh.ClientConfig{
		User: Config.BTCServerUser,
		Auth: []ssh.AuthMethod{ssh.Password(Config.BTCServerPwd)},
		HostKeyCallback: func(hostname string, remote net.Addr, key ssh.PublicKey) error {
			return nil
		},
	}
	sshConfig.HostKeyCallback = ssh.InsecureIgnoreHostKey()
	url, _ := url.Parse(strings.Join([]string{"http://", Config.BTCRPCHTTP}, ""))
	client, err := ssh.Dial("tcp", strings.Join([]string{url.Hostname(), "22"}, ":"), sshConfig)
	if err != nil {
		return nil, errors.New(strings.Join([]string{"ssh dial error: ", err.Error()}, ""))
	}
  return client, nil
}

// Remote2local copy file to local
func Remote2local(dstFileWithPath, srcFileWithPath string)  {
  sshClient, err := NewSSHClient()
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

  dstFile, err := os.Create(dstFileWithPath)
  if err != nil {
    Sugar.Fatal("Create dst file error: ", err.Error())
  }

  if _, err := io.Copy(dstFile, srcFile); err != nil {
    Sugar.Fatal("io copy error: ", err.Error())
  }

  if err := dstFile.Sync(); err != nil {
    Sugar.Fatal("dsfFile error: ", err.Error())
  }

}
