package db

import (
	"bufio"
	"github.com/syndtr/goleveldb/leveldb"
	"os"
	"errors"
	"strings"
	"syscall"
	"wallet-transition/pkg/configure"
	"wallet-transition/pkg/util"
)

// LDB level db
type LDB struct {
	*leveldb.DB
}

// NewLDB new leveldb
func NewLDB(asset string) (*LDB, error) {
	dir := strings.Join([]string{util.HomeDir(), configure.Config.DBWalletPath, asset}, "/")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, errors.New(strings.Join([]string{"NewLDB error: ", err.Error()}, ""))
	}

	db, err := leveldb.OpenFile(dir, nil)
	if err != nil {
		return nil, errors.New(strings.Join([]string{"NewLDB error: ", err.Error()}, ""))
	}
	return &LDB{db}, nil
}

// MigrateBTC migrate btc wallet to lleveldb
func (db *LDB) MigrateBTC() {
	file, err := os.Open(strings.Join([]string{configure.Config.BackupWalletPath, "btc.backup"}, ""))
	if err != nil {
		configure.Sugar.Fatal("open dump wallet file error: ", err.Error())
	}

	defer syscall.Umask(syscall.Umask(0))
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.Contains(line, "# addr=") && !strings.Contains(line, "script") {
			splitArr := strings.Split(line, " ")
			privateKey := splitArr[0]
			addressStr := strings.Split(splitArr[4], "=")[1]

			addresses := strings.Split(addressStr, ",")
			for _, address := range addresses {
				_, err := db.Get([]byte(address), nil)
				if err != nil && strings.Contains(err.Error(), "leveldb: not found") {
					db.Put([]byte(address), []byte(privateKey), nil)
					configure.Sugar.Info("successful migrated ", address)
				}else if err != nil {
					configure.Sugar.Fatal("Failt to migrate: ", address)
				}else {
					configure.Sugar.Info("Exists in db, skip ", address)
				}
			}
		}
	}
	if err := scanner.Err(); err != nil {
		configure.Sugar.Fatal("scanner error: ", err.Error())
	}
}
