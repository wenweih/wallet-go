package db

import (
  "os"
  "errors"
  "strings"
  "path/filepath"
  "github.com/qor/transition"
  "github.com/jinzhu/gorm"
  "bytes"
  // sqlite driven
  _ "github.com/jinzhu/gorm/dialects/sqlite"
  _ "github.com/jinzhu/gorm/dialects/mysql"
  "wallet-transition/pkg/configure"
  "github.com/btcsuite/btcd/btcjson"
)

// NewSqlite new sqlite3 connection
func NewSqlite() (*GormDB, error) {
  if err := os.MkdirAll(filepath.Dir(configure.Config.BackupWalletPath), 0700); err != nil {
    return nil, errors.New(strings.Join([]string{"MkdirAll error: ", err.Error()}, ""))
  }

  db, err := gorm.Open("sqlite3", strings.Join([]string{configure.Config.BackupWalletPath, "wallet-transition.db"}, ""))
  if err != nil {
    return nil, errors.New(strings.Join([]string{"failed to connect database:", err.Error()}, ""))
  }
  db.AutoMigrate(&SubAddress{}, &BTCBlock{}, &UTXO{}, &transition.StateChangeLog{})
  return &GormDB{db}, nil
}

// NewMySQL new mysql connection
func NewMySQL() (*GormDB, error) {
  w := bytes.Buffer{}
  dbConfig := configure.Config.MySQLUser + ":" + configure.Config.MySQLPass + "@tcp(" + configure.Config.MySQLHost + ")/" + configure.Config.MySQLName
	w.WriteString(dbConfig)
	w.WriteString("?charset=utf8&parseTime=True&loc=Local")
	dbInfo := w.String()
	db, err := gorm.Open("mysql", dbInfo)
  if err != nil {
    return nil, errors.New(strings.Join([]string{"failed to connect database:", err.Error()}, ""))
  }
  db.AutoMigrate(&SubAddress{}, &BTCBlock{}, &UTXO{}, &transition.StateChangeLog{})
  return &GormDB{db}, nil
}

// GetBTCBestBlockOrCreate get btc best block in sqlite
func (db *GormDB) GetBTCBestBlockOrCreate(block *btcjson.GetBlockVerboseResult) (*BTCBlock, error) {
  var bestBlock BTCBlock
  err := db.Order("height desc").First(&bestBlock).Error
  if err != nil && err.Error() == "record not found" {
    configure.Sugar.Info("best block info not found in btc_blocks table, init ....")
    bestBlock.Hash = block.Hash
    bestBlock.Height = block.Height
    if err = db.BlockInfo2DB(bestBlock, block); err != nil {
      return nil, err
    }
  } else if err != nil {
    return nil, errors.New(strings.Join([]string{"Get bestBlock error: ", err.Error()}, ""))
  }
  return &bestBlock, nil
}

// BlockInfo2DB iterator each block tx
func (db *GormDB) BlockInfo2DB(dbBlock BTCBlock, rawBlock *btcjson.GetBlockVerboseResult) error {
  ts := db.Begin()
  if err := ts.Create(&dbBlock).Error; err != nil {
    if err = ts.Rollback().Error; err != nil {
      return errors.New(strings.Join([]string{"database rollback error: create bestblock record ", err.Error()}, ""))
    }
  }
  for _, tx := range rawBlock.Tx {
    for _, vout := range tx.Vout {
      for _, address := range vout.ScriptPubKey.Addresses {
        var addrDB SubAddress
        if err := db.Where("address = ? AND asset = ?", address, "btc").First(&addrDB).Error; err != nil && err.Error() == "record not found" {
          continue
        }else if err != nil {
          configure.Sugar.DPanic(strings.Join([]string{"query sub address err: ", address, " ", err.Error()}, ""))
        }
        utxo := UTXO{Txid: tx.Txid, Amount: vout.Value, Height: rawBlock.Height, VoutIndex: vout.N, SubAddress: addrDB, BTCBlock: dbBlock}
        utxo.SetState("original")
        if err := ts.Create(&utxo).Error; err != nil {
          if err := ts.Rollback().Error; err != nil {
            return errors.New(strings.Join([]string{"database rollback error: create utxo record ", err.Error()}, ""))
          }
        }
        configure.Sugar.Info("create utxo: ", "address=", addrDB.Address, " utxoID=", utxo.ID)
      }
    }
  }
  if err := ts.Commit().Error; err != nil {
    if err = ts.Rollback().Error; err != nil {
      return errors.New(strings.Join([]string{"database rollback error: create utxo record ", err.Error()}, ""))
    }
    return errors.New(strings.Join([]string{"database commit error: ", err.Error()}, ""))
  }
  configure.Sugar.Info("Finish BlockInfo2DB: ", rawBlock.Height, " ", rawBlock.Hash)
  return nil
}

// RollbackTrack rollback 6 blocks when add a new btc_block
func (db *GormDB) RollbackTrackBTC(bestHeight int64, backTracking bool, rawBlock *btcjson.GetBlockVerboseResult) (bool, int64) {
  trackHeight := rawBlock.Height
  var (
    dbBlock BTCBlock
    utxos []UTXO
  )
  if err := db.First(&dbBlock, "height = ? AND re_org = ?", rawBlock.Height, false).Related(&utxos).Error; err !=nil && err.Error() == "record not found" {
    dbBlock.Hash = rawBlock.Hash
    dbBlock.Height = rawBlock.Height
    if err = db.BlockInfo2DB(dbBlock, rawBlock); err != nil {
      configure.Sugar.Fatal(err.Error())
    }
  }else if err != nil {
    configure.Sugar.Fatal("Find track block error:", err.Error())
  }else {
    if dbBlock.Hash != rawBlock.Hash {
      ts := db.Begin()
      // update utxos related with the dbBlock
      ts.Model(&dbBlock).Update("re_org", true)
      for _, utxo := range utxos {
        ts.Model(&utxo).Update("re_org", true)
      }
      ts.Commit()
      if err = db.BlockInfo2DB(BTCBlock{Hash: rawBlock.Hash, Height: rawBlock.Height}, rawBlock); err != nil {
        configure.Sugar.Fatal(err.Error())
      }
      configure.Sugar.Info("reorg:", dbBlock.Height, " ", dbBlock.Hash)
    } else {
      configure.Sugar.Info("tracking the same block, nothing happen ", dbBlock.Height, " ", dbBlock.Hash)
    }
  }

  if trackHeight < bestHeight - 5 {
    backTracking = false
  }else {
    backTracking = true
    trackHeight --
  }
  return backTracking, trackHeight
}
