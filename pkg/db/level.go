package db

import (
	"bufio"
	"encoding/hex"
	"github.com/syndtr/goleveldb/leveldb"
	"os"
	"errors"
	"strings"
	"syscall"
	"wallet-transition/pkg/configure"
)

// LDB level db
type LDB struct {
	*leveldb.DB
}

const (
  // BitcoinCoreLD bitcoin private key folder name
  BitcoinCoreLD string = "btc"
  // EthereumLD ethereum private key folder name
  EthereumLD    string = "eth"
)

// NewLDB new leveldb
func NewLDB(asset string) (*LDB, error) {
	dir := strings.Join([]string{configure.HomeDir(), configure.Config.DBWalletPath, asset}, "/")
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
	sqldb, err := NewMySQL()
	if err != nil {
		configure.Sugar.Fatal("NewMySQL error: ", err.Error())
	}
	defer sqldb.Close()

	file, err := os.Open(strings.Join([]string{configure.Config.BackupWalletPath, "btc.backup_new"}, ""))
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
					configure.Sugar.Info("Exists in level db, skip ", address)
				}
				var subAddress SubAddress
				if err := sqldb.Where(SubAddress{Address: address, Asset: "btc"}).FirstOrCreate(&subAddress).Error; err != nil {
					configure.Sugar.Fatal("insert address to db error: ", err.Error())
				}
			}
		}
	}
	if err := scanner.Err(); err != nil {
		configure.Sugar.Fatal("scanner error: ", err.Error())
	}
}

// MigrateETH migrate eth wallet to lleveldb
func (db *LDB) MigrateETH ()  {
	sqldb, err := NewMySQL()
	if err != nil {
		configure.Sugar.Fatal("NewMySQL error: ", err.Error())
	}
	defer sqldb.Close()

	file, err := os.Open(strings.Join([]string{configure.Config.BackupWalletPath, "eth.backup_new"}, ""))
	if err != nil {
		configure.Sugar.Fatal("open dump wallet file error: ", err.Error())
	}

	defer syscall.Umask(syscall.Umask(0))
	defer file.Close()

	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		splitArr := strings.Split(scanner.Text(), " ")
		address := strings.ToLower(splitArr[0])
		address = strings.ToLower(address)
		priv, err := hex.DecodeString(splitArr[1])
		if err != nil {
			configure.Sugar.Fatal("Decode priv error")
		}

		_, err = db.Get([]byte(address), nil)
		if err != nil && strings.Contains(err.Error(), "leveldb: not found") {
			db.Put([]byte(address), priv, nil)
			configure.Sugar.Info("successful migrated ", address)
		}else if err != nil {
			configure.Sugar.Fatal("Failt to migrate: ", address)
		}else {
			if err := db.Put([]byte(address), priv, nil); err != nil {
				configure.Sugar.Fatal("Exists in db, fail to override ", address)
			}
			configure.Sugar.Info("Exists in db, override ", address)
		}

		var subAddress SubAddress
		if err := sqldb.Where(SubAddress{Address: address, Asset: "eth"}).FirstOrCreate(&subAddress).Error; err != nil {
			configure.Sugar.Fatal("insert address to db error: ", err.Error())
		}
	}
	if err := scanner.Err(); err != nil {
		configure.Sugar.Fatal("scanner error: ", err.Error())
	}
}
