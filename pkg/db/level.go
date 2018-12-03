package db

import (
	"bufio"
	"github.com/syndtr/goleveldb/leveldb"
	"os"
	"strings"
	"syscall"
	"wallet-transition/pkg/configure"
	"wallet-transition/pkg/util"
)

// BTCMigrate migrate btc wallet to lleveldb
func BTCMigrate() {
	file, err := os.Open(configure.Config.NewBTCWalletFileName)
	if err != nil {
		configure.Sugar.Fatal("open dump wallet file error: ", err.Error())
	}
	defer file.Close()

	defer syscall.Umask(syscall.Umask(0))
	dir := strings.Join([]string{util.HomeDir(), ".db_wallet/btc"}, "/")
	if err = os.MkdirAll(dir, 0755); err != nil {
		configure.Sugar.Fatal("mkdir btc db wallet error: ", err.Error())
	}

	db, err := leveldb.OpenFile(dir, nil)
	if err != nil {
		configure.Sugar.Fatal("open btc wallet error: ", err.Error())
	}
	defer db.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.Contains(line, "hdkeypath") {
			splitArr := strings.Split(line, " ")
			privateKey := splitArr[0]
			address := strings.Split(splitArr[4], "=")[1]

			_, err := db.Get([]byte(address), nil)
			if err != nil && strings.Contains(err.Error(), "leveldb: not found") {
				db.Put([]byte(address), []byte(privateKey), nil)
				configure.Sugar.Info("successful migrated ", address)
			}
			if err != nil && !strings.Contains(err.Error(), "leveldb: not found") {
				configure.Sugar.Fatal("Failt to migrate: ", address)
			}
		}
	}
	if err := scanner.Err(); err != nil {
		configure.Sugar.Fatal("scanner error: ", err.Error())
	}
}
