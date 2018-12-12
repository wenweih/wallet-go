package db

import (
  "os"
  "errors"
  "strings"
  "path/filepath"
  "github.com/jinzhu/gorm"
  // sqlite driven
  _ "github.com/jinzhu/gorm/dialects/sqlite"
  "wallet-transition/pkg/configure"
)

// GormDB relation database
type GormDB struct {
	*gorm.DB
}

// SubAddress 监听地址
type SubAddress struct {
	gorm.Model
	Address string `gorm:"type:varchar(42);not null;unique_index"`
  Asset   string `gorm:"type:varchar(42);not null"`
  UTXOs   []UTXO
}

// BTCBlock notify block info
type BTCBlock struct {
  gorm.Model
  Hash    string `gorm:"not null;unique_index"`
  Height  int64   `gorm:"not null"`
}

// UTXO utxo model
type UTXO struct {
  gorm.Model
  Txid          string    `gorm:"not null"`
  Amount        float64   `gorm:"not null"`
  Height        int64     `gorm:"not null"`
  VoutIndex     uint32    `gorm:"not null"`
  SubAddress    SubAddress
  SubAddressID  int
  BTCBlock      BTCBlock
  BTCBlockID    int
}

// NewSqlite new sqlite3 connection
func NewSqlite() (*GormDB, error) {
  if err := os.MkdirAll(filepath.Dir(configure.Config.BackupWalletPath), 0700); err != nil {
    return nil, errors.New(strings.Join([]string{"MkdirAll error: ", err.Error()}, ""))
  }

  db, err := gorm.Open("sqlite3", strings.Join([]string{configure.Config.BackupWalletPath, "wallet-transition.db"}, ""))
  if err != nil {
    return nil, errors.New(strings.Join([]string{"failed to connect database:", err.Error()}, ""))
  }
  db.AutoMigrate(&SubAddress{}, &BTCBlock{}, &UTXO{})
  return &GormDB{db}, nil
}

// GetBTCBestBlockOrCreate get btc best block in sqlite
func (db *GormDB) GetBTCBestBlockOrCreate(hash string, height int64) (*BTCBlock, error) {
  var bestBlock BTCBlock
  err := db.Order("height desc").First(&bestBlock).Error
  if err != nil && err.Error() == "record not found" {
    configure.Sugar.Info("best block info not found in btc_blocks table, init ....")
    bestBlock.Hash = hash
    bestBlock.Height = height
    db.Create(&bestBlock)
  } else if err != nil {
    return nil, errors.New(strings.Join([]string{"Get bestBlock error: ", err.Error()}, ""))
  }
  return &bestBlock, nil
}
