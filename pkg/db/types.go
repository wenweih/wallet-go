package db

import (
  "github.com/jinzhu/gorm"
  "github.com/qor/transition"
  "github.com/syndtr/goleveldb/leveldb"
)

// LDB level db
type LDB struct {
	*leveldb.DB
}

// GormDB relation database
type GormDB struct {
	*gorm.DB
}

// UTXO utxo model
type UTXO struct {
  gorm.Model
  Txid                  string    `gorm:"not null"`
  Amount                float64   `gorm:"not null"`
  Height                int64     `gorm:"not null"`
  VoutIndex             uint32    `gorm:"not null"`
  ReOrg                 bool      `gorm:"not null;default:false"`
  UsedBy                string
  Chain                 string
  SubAddress            SubAddress
  SubAddressID          uint
  SimpleBitcoinBlock    SimpleBitcoinBlock
  SimpleBitcoinBlockID  uint
  transition.Transition
}

// SubAddress 监听地址
type SubAddress struct {
	gorm.Model
	Address string `gorm:"type:varchar(100);not null;unique_index"`
  Asset   string `gorm:"type:varchar(42);not null"`
  UTXOs   []UTXO
}

// SimpleBitcoinBlock notify block info
type SimpleBitcoinBlock struct {
  gorm.Model
  Hash    string  `gorm:"not null;unique_index:idx_hash_height"`
  Height  int64   `gorm:"not null;unique_index:idx_hash_height"`
  UTXOs   []UTXO  `gorm:"foreignkey:SimpleBitcoinBlockID;association_foreignkey:Refer"`
  ReOrg   bool    `gorm:"default:false"`
  Chain    string
}
