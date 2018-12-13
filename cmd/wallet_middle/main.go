package main

import (
  "errors"
  "strings"
  "net/http"
  "github.com/gin-gonic/gin"
  "wallet-transition/pkg/db"
  "wallet-transition/pkg/util"
  "wallet-transition/pkg/configure"
  "wallet-transition/pkg/blockchain"
  "github.com/btcsuite/btcd/chaincfg/chainhash"
)

var (
  btcClient *blockchain.BTCRPC
  sqldb *db.GormDB
)

func main() {
  var err error
  btcClient = &blockchain.BTCRPC{Client: blockchain.NewbitcoinClient()}
  sqldb, err = db.NewSqlite()
  if err != nil {
    configure.Sugar.Fatal(err.Error())
  }
  defer sqldb.Close()

  binfo, err := btcClient.Client.GetBlockChainInfo()
  if err != nil {
    configure.Sugar.Fatal("GetBlockChainInfo error:", err.Error())
  }
  rollbackBlock, err := btcClient.GetBlock(int64(binfo.Headers - 5))

  bestBlock, err := sqldb.GetBTCBestBlockOrCreate(rollbackBlock)
  if err != nil {
    configure.Sugar.Fatal(err.Error())
  }
  configure.Sugar.Info("DB bestBlock is: ", bestBlock.Height, " ", bestBlock.Hash, " Chain bestBlock is: ", binfo.Headers, " ", binfo.BestBlockHash)

  dbBestHeight := bestBlock.Height
  for height := (dbBestHeight + 1); height <= int64(binfo.Headers); height++ {
    rawBlock, err := btcClient.GetBlock(height)
    if err != nil {
      configure.Sugar.Fatal(err.Error())
    }
    dbBlock := db.BTCBlock{Hash: rawBlock.Hash, Height: rawBlock.Height}
    if err = sqldb.BlockInfo2DB(dbBlock, rawBlock); err != nil {
      configure.Sugar.Fatal(err.Error())
    }
  }

  gin.SetMode(gin.ReleaseMode)
  r := gin.Default()
  r.GET("/btc-best-block-notify", btcBestBlockNotifyHandle)
  if err := r.Run(":3001"); err != nil {
    configure.Sugar.Fatal(err.Error())
  }
}

func btcBestBlockNotifyHandle(c *gin.Context) {
  strHash := c.Query("hash")
  blockHash, err := chainhash.NewHashFromStr(strHash)
  if err != nil {
    configure.Sugar.DPanic(err.Error())
    util.GinRespException(c, http.StatusInternalServerError, errors.New(strings.Join([]string{"NewHashFromStr:", err.Error()}, "")))
    return
  }
  rawBlock, err := btcClient.Client.GetBlockVerboseTxM(blockHash)
  if err != nil {
    configure.Sugar.DPanic(err.Error())
    util.GinRespException(c, http.StatusInternalServerError, errors.New(strings.Join([]string{"GetBlockVerboseTxM:", err.Error()}, "")))
    return
  }

  dbBlock := db.BTCBlock{Hash: rawBlock.Hash, Height: rawBlock.Height}
  if err = sqldb.BlockInfo2DB(dbBlock, rawBlock); err != nil {
    configure.Sugar.Fatal(err.Error())
  }

  backTracking := true
  trackHeight := rawBlock.Height - 1
  for backTracking {
    block, err := btcClient.GetBlock(trackHeight)
    if err !=nil {
      configure.Sugar.Fatal("backTracking error:", err.Error())
    }

    var (
      dbBlock db.BTCBlock
      utxos []db.UTXO
    )
    if err = sqldb.Where("height = ? AND re_org = ?", block.Height, false).First(&dbBlock).Related(&utxos).Error; err !=nil && err.Error() == "record not found" {
      dbBlock.Hash = block.Hash
      dbBlock.Height = block.Height
      if err = sqldb.BlockInfo2DB(dbBlock, block); err != nil {
        configure.Sugar.Fatal(err.Error())
      }
      backTracking = true
      trackHeight --
    }else if err != nil {
      configure.Sugar.Fatal("Find track block error:", err.Error())
    }else {
      if dbBlock.Hash != block.Hash {
        ts := sqldb.Begin()
        // update utxos related with the dbBlock
        ts.Model(&dbBlock).Update("re_org", true)
        for _, utxo := range utxos {
          ts.Model(&utxo).Update("re_org", true)
        }
        ts.Commit()
        if err = sqldb.BlockInfo2DB(db.BTCBlock{Hash: block.Hash, Height: block.Height}, block); err != nil {
          configure.Sugar.Fatal(err.Error())
        }
        configure.Sugar.Info("reorg:", dbBlock.Height, " ", dbBlock.Hash)
      } else {
        configure.Sugar.Info("tracking the same block, nothing happen ", dbBlock.Height, " ", dbBlock.Hash)
      }
      if trackHeight < rawBlock.Height - 5 {
        backTracking = false
      }else {
        trackHeight --
      }
    }
  }

  c.JSON(http.StatusOK, gin.H {
    "status": http.StatusOK,
    "address": "hi",
  })
}
