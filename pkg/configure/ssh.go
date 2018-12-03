package configure

import (
	"errors"
	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
	"io"
	"net"
	"os"
	"path/filepath"
	"strings"
	"wallet-transition/pkg/util"
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
func (c *ServerClient) CopyRemoteFile2(srcFileWithPath, dstFileWithPath string, local bool) {
	// http://networkbit.ch/golang-sftp-client/
	srcFile, err := c.SftpClient.Open(srcFileWithPath)
	if err != nil {
		Sugar.Fatal("Open src file error: ", err.Error())
	}
	if local {
		path := strings.Join([]string{util.HomeDir(), filepath.Base(dstFileWithPath)}, "/")
		dstFile, err := os.Create(path)
		if err != nil {
			Sugar.Fatal("Create dst file error: ", err.Error())
		}
		if _, err := io.Copy(dstFile, srcFile); err != nil {
			Sugar.Fatal("io copy error: ", err.Error())
		}
		if err := dstFile.Sync(); err != nil {
			Sugar.Fatal("dsfFile error: ", err.Error())
		}
		defer dstFile.Close()
		Sugar.Info("Copy to local: ", path)
	} else {
		newWalletServerClient, err := NewServerClient(Config.NewBTCWalletServerUser, Config.NewBTCWalletServerPass, Config.NewBTCWalletServerHost)
		if err != nil {
			Sugar.Fatal(err.Error())
		}

		// create folder for old wallet backup
		if err = newWalletServerClient.SftpClient.MkdirAll(filepath.Dir(dstFileWithPath)); err != nil {
			Sugar.Fatal(err.Error())
		}
		dstFile, err := newWalletServerClient.SftpClient.Create(Config.NewBTCWalletFileName)
		defer dstFile.Close()

		if _, err := io.Copy(dstFile, srcFile); err != nil {
			Sugar.Fatal("Copy dst file error: ", err.Error())
		}

		if err := newWalletServerClient.Close(); err != nil {
			Sugar.Fatal(err.Error())
		}
		Sugar.Info("Copy to new server: ", newWalletServerClient.SSHClient.RemoteAddr().String(), ":", dstFileWithPath)
	}

	defer srcFile.Close()
	if err := c.Close(); err != nil {
		Sugar.Fatal(err.Error())
	}
}
