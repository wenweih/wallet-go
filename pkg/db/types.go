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

// SubAddress 监听地址
type SubAddress struct {
	gorm.Model
	Address string `gorm:"type:varchar(100);not null;unique_index"`
  Asset   string `gorm:"type:varchar(42);not null"`
  UTXOs   []UTXO
}

// Block notify block info
type Block struct {
  gorm.Model
  Hash    string `gorm:"not null;index"`
  Height  int64   `gorm:"not null"`
  UTXOs   []UTXO
  ReOrg   bool    `gorm:"default:false"`
  Chain    string
}

// UTXO utxo model
type UTXO struct {
  gorm.Model
  Txid          string    `gorm:"not null"`
  Amount        float64   `gorm:"not null"`
  Height        int64     `gorm:"not null"`
  VoutIndex     uint32    `gorm:"not null"`
  ReOrg         bool      `gorm:"not null;default:false"`
  SubAddress    SubAddress
  UsedBy        string
  SubAddressID  uint
  Block         Block
  BTCBlockID    uint
  transition.Transition
}
