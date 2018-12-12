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
  Asset  string `gorm:"type:varchar(42);not null"`
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
  db.AutoMigrate(&SubAddress{})
  return &GormDB{db}, nil
}
