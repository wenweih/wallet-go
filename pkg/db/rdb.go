package db

import (
  "os"
  "fmt"
  "errors"
  "strings"
  "path/filepath"
  "github.com/qor/transition"
  "github.com/jinzhu/gorm"
  "bytes"
  // sqlite driven
  _ "github.com/jinzhu/gorm/dialects/sqlite"
  _ "github.com/jinzhu/gorm/dialects/mysql"
  "wallet-go/pkg/configure"
  "wallet-go/pkg/common"
  "github.com/btcsuite/btcd/btcjson"
)

// NewSqlite new sqlite3 connection
func NewSqlite() (*GormDB, error) {
  if err := os.MkdirAll(filepath.Dir(configure.Config.BackupWalletPath), 0700); err != nil {
    return nil, errors.New(strings.Join([]string{"MkdirAll error: ", err.Error()}, ""))
  }

  db, err := gorm.Open("sqlite3", strings.Join([]string{configure.Config.BackupWalletPath, "wallet-go.db"}, ""))
  if err != nil {
    return nil, errors.New(strings.Join([]string{"failed to connect database:", err.Error()}, ""))
  }
  db.AutoMigrate(&SubAddress{}, &SimpleBitcoinBlock{}, &UTXO{}, &transition.StateChangeLog{})
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
  configure.Sugar.Info("database connecting...")
  db.AutoMigrate(&SubAddress{}, &SimpleBitcoinBlock{}, &UTXO{}, &transition.StateChangeLog{})
  db.DB().SetMaxIdleConns(100)
  return &GormDB{db}, nil
}

// GetBTCBestBlockOrCreate get btc best block in sqlite
func (db *GormDB) GetBTCBestBlockOrCreate(block *btcjson.GetBlockVerboseResult, chain string) (*SimpleBitcoinBlock, error) {
  var bestBlock SimpleBitcoinBlock
  err := db.Order("height desc").First(&bestBlock).Error
  if err != nil && err.Error() == "record not found" {
    configure.Sugar.Info("best block info not found in btc_blocks table, init ....")
    bestBlock.Hash = block.Hash
    bestBlock.Height = block.Height
    if err = db.BlockInfo2DB(bestBlock, block, chain); err != nil {
      return nil, err
    }
  } else if err != nil {
    return nil, errors.New(strings.Join([]string{"Get bestBlock error: ", err.Error()}, ""))
  }
  return &bestBlock, nil
}

// CreateBitcoinBlockWithUTXOs save block and utxo related with subAddress blockResultCh <-chan
func (db *GormDB) CreateBitcoinBlockWithUTXOs(queryBlockResultCh <- chan common.QueryBlockResult) (<-chan common.CreateBlockResult) {
  createBlockCh := make(chan common.CreateBlockResult)
  go func(db *GormDB) {
    defer close(createBlockCh)
    var (
      rawBlock *btcjson.GetBlockVerboseResult
      chain string
    )
    for b := range queryBlockResultCh {
      if b.Error != nil {
        createBlockCh <- common.CreateBlockResult{Error: b.Error}
        return
      }
      rawBlock = b.Block.(*btcjson.GetBlockVerboseResult)
      chain = b.Chain
    }

    var block SimpleBitcoinBlock
    ts := db.Begin()
    if err := ts.FirstOrCreate(&block, SimpleBitcoinBlock{
      Hash: rawBlock.Hash,
      Height: rawBlock.Height,
      Chain: chain,
    }).Error; err != nil {
      ts.Rollback()
      createBlockCh <- common.CreateBlockResult{Error: fmt.Errorf("create block error: %s", err)}
      return
    }

    for _, tx := range rawBlock.Tx {
      for _, vout := range tx.Vout {
        for _, address := range vout.ScriptPubKey.Addresses {
          var (
            addr SubAddress
            utxo UTXO
          )
          if err := db.Where("address = ? AND asset = ?", address, chain).First(&addr).Error; err != nil && err.Error() == "record not found" {
            continue
          }else if err != nil {
            createBlockCh <- common.CreateBlockResult{Error: fmt.Errorf("Query sub address err: %s", err)}
            return
          }
          ts.FirstOrCreate(&utxo, UTXO{Txid: tx.Txid,
            Amount: vout.Value,
            Height: rawBlock.Height,
            VoutIndex: vout.N,
            SubAddressID: addr.ID})
        }
      }
    }
    if err := ts.Commit().Error; err != nil {
      ts.Rollback()
      createBlockCh <- common.CreateBlockResult{Error: fmt.Errorf("database transaction err: %s", err)}
      return
    }
    configure.Sugar.Info("Saved block to database, height: ", block.Height, " hash: ", block.Hash)
    createBlockCh <- common.CreateBlockResult{Block: block}
  }(db)
  return createBlockCh
}

// BlockInfo2DB iterator each block tx
func (db *GormDB) BlockInfo2DB(dbBlock SimpleBitcoinBlock, rawBlock *btcjson.GetBlockVerboseResult, chain string) error {
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
        if err := db.Where("address = ? AND asset = ?", address, chain).First(&addrDB).Error; err != nil && err.Error() == "record not found" {
          continue
        }else if err != nil {
          configure.Sugar.DPanic(strings.Join([]string{"query sub address err: ", address, " ", err.Error()}, ""))
        }
        utxo := UTXO{Txid: tx.Txid, Amount: vout.Value, Height: rawBlock.Height, VoutIndex: vout.N, SubAddress: addrDB, SimpleBitcoinBlock: dbBlock}
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

// TrackBlock rollback 6 blocks when save new block records
func (db *GormDB) TrackBlock(bestBlockHeight int64, isTracking bool, queryBlockResultCh <- chan common.QueryBlockResult) (bool, int64) {
  var (
    rawBlock *btcjson.GetBlockVerboseResult
    chain string
    dbBlock SimpleBitcoinBlock
    utxos []UTXO
  )


  for b := range queryBlockResultCh {
    if b.Error != nil {
      configure.Sugar.Fatal(b.Error.Error())
    }
    rawBlock = b.Block.(*btcjson.GetBlockVerboseResult)
    chain = b.Chain
  }

  trackHeight := rawBlock.Height

  if err := db.First(&dbBlock, "height = ? AND re_org = ?", rawBlock.Height, false).Related(&utxos).Error; err !=nil && err.Error() == "record not found" {
    dbBlock.Hash = rawBlock.Hash
    dbBlock.Height = rawBlock.Height
    blockCh := make(chan common.QueryBlockResult)
    go func (rawBlock *btcjson.GetBlockVerboseResult)  {
      defer close(blockCh)
      blockCh  <- common.QueryBlockResult{Block: rawBlock, Chain: chain}
    }(rawBlock)
    createBlockResul := <- db.CreateBitcoinBlockWithUTXOs(blockCh)
    if createBlockResul.Error != nil{
      configure.Sugar.Fatal(createBlockResul.Error.Error())
    }

    if createBlockResul.Error == nil{
      bestBlock := createBlockResul.Block.(SimpleBitcoinBlock)
      configure.Sugar.Info("create block successfully,", " height: ", bestBlock.Height, " hash: ", bestBlock.Hash)
    }
  }else if err != nil {
    configure.Sugar.Fatal("query track block error:", err.Error())
  }else {
    if dbBlock.Hash != rawBlock.Hash {
      ts := db.Begin()
      // update utxos related with the dbBlock
      ts.Model(&dbBlock).Update("re_org", true)
      for _, utxo := range utxos {
        ts.Model(&utxo).Update("re_org", true)
      }
      ts.Commit()

      blockCh := make(chan common.QueryBlockResult)
      blockCh  <- common.QueryBlockResult{Block: rawBlock, Chain: chain}
      createBlockResul := <- db.CreateBitcoinBlockWithUTXOs(blockCh)
      close(blockCh)
      if createBlockResul.Error == nil{
        bestBlock := createBlockResul.Block.(SimpleBitcoinBlock)
        configure.Sugar.Info("create block successfully,", " height: ", bestBlock.Height, " hash: ", bestBlock.Hash)
      }
      configure.Sugar.Info("reorg:", dbBlock.Height, " ", dbBlock.Hash)
    } else {
      configure.Sugar.Info("tracking the same block, nothing happen ", dbBlock.Height)
    }
  }

  if trackHeight < bestBlockHeight - 5 {
    isTracking = false
  }else {
    isTracking = true
    trackHeight --
  }
  return isTracking, trackHeight
}

// RollbackTrackBTC rollback 6 blocks when add a new btc_block
func (db *GormDB) RollbackTrackBTC(bestHeight int64, backTracking bool, rawBlock *btcjson.GetBlockVerboseResult, chain string) (bool, int64) {
  trackHeight := rawBlock.Height
  var (
    dbBlock SimpleBitcoinBlock
    utxos []UTXO
  )
  if err := db.First(&dbBlock, "height = ? AND re_org = ?", rawBlock.Height, false).Related(&utxos).Error; err !=nil && err.Error() == "record not found" {
    dbBlock.Hash = rawBlock.Hash
    dbBlock.Height = rawBlock.Height
    if err = db.BlockInfo2DB(dbBlock, rawBlock, chain); err != nil {
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
      if err = db.BlockInfo2DB(SimpleBitcoinBlock{Hash: rawBlock.Hash, Height: rawBlock.Height}, rawBlock, chain); err != nil {
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
