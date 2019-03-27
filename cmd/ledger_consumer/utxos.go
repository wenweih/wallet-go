package main

import (
  "sync"
  "encoding/json"
  "github.com/spf13/cobra"
  "wallet-go/pkg/db"
  "wallet-go/pkg/mq"
  "wallet-go/pkg/common"
  "wallet-go/pkg/configure"
  "wallet-go/pkg/blockchain"
  "github.com/streadway/amqp"
  "github.com/btcsuite/btcd/btcjson"
)

var (
  err error
  messageClient mq.IMessagingClient
  sqldb *db.GormDB
  b *blockchain.Blockchain
)

// UTXO command
var UTXO = &cobra.Command {
  Use:   "utxo",
  Short: "ledger consumer, maintain utxos",
  Run: func (cmd *cobra.Command, args []string) {
    switch chain {
    case "bitcoincore":
      bitcoinClient, err := blockchain.NewbitcoinClient()
      if err != nil {
        configure.Sugar.Fatal(err.Error())
      }
      sqldb, err = db.NewMySQL()
      if err != nil {
        configure.Sugar.Fatal(err.Error())
      }
      defer sqldb.Close()
      chain := blockchain.BitcoinCoreChain{Client: bitcoinClient}
      b = blockchain.NewBlockchain(nil, nil, chain)

      // query ledger info
      ledgerInfoI, err := b.Query.Ledger()
      if err != nil {
        configure.Sugar.Fatal(err)
      }
      ledgerInfo := ledgerInfoI.(*btcjson.GetBlockChainInfoResult)

      queryBlockResult := b.Query.Block(int64(ledgerInfo.Headers - 5))
      createBlockResul := <- sqldb.CreateBitcoinBlockWithUTXOs(queryBlockResult)
      if createBlockResul.Error != nil{
        configure.Sugar.Fatal(createBlockResul.Error.Error())
      }

      bestBlock := createBlockResul.Block.(db.SimpleBitcoinBlock)
      configure.Sugar.Info("create block successfully,", " height: ", bestBlock.Height, " hash: ", bestBlock.Hash)

      isTracking := true
      trackHeight := bestBlock.Height - 1
      for isTracking {
        ch := b.Query.Block(trackHeight)
        isTracking, trackHeight = sqldb.TrackBlock(bestBlock.Height, isTracking, ch)
      }

      dbBestHeight := bestBlock.Height
      for height := (dbBestHeight + 1); height <= int64(ledgerInfo.Headers); height++ {
        ch := b.Query.Block(height)
        createBlockResul := <- sqldb.CreateBitcoinBlockWithUTXOs(ch)
        if createBlockResul.Error != nil{
          configure.Sugar.Fatal(createBlockResul.Error.Error())
        }
      }

      var wg sync.WaitGroup
      wg.Add(1)
      go bitcoinMQ(&wg, )
      wg.Wait()
    default:
      configure.Sugar.Fatal("Unsupport chain: ", chain)
    }
  },
}

func bitcoinMQ(wg *sync.WaitGroup)  {
  defer wg.Done()
  forever := make(chan bool)
  messageClient = &mq.MessagingClient{}
  messageClient.ConnectToBroker(configure.Config.MQ)
  if err := messageClient.Subscribe("bestblock", "fanout", "bitcoincore_best_block_queue", "bitcoincore", "", onBitcoinMessage); err != nil {
    configure.Sugar.Fatal("monitor address mq subscribe error: ", err.Error())
  }
  <-forever
}

func onBitcoinMessage(d amqp.Delivery) {
	var mqdata *btcjson.GetBlockVerboseResult
	if err := json.Unmarshal(d.Body, &mqdata); err != nil {
    configure.Sugar.DPanic(err.Error())
  }
  configure.Sugar.Info("consumer bitcoin new block: ", mqdata.Hash, mqdata.Height)

  blockCh := make(chan common.QueryBlockResult)
  go func (rawBlock *btcjson.GetBlockVerboseResult)  {
    defer close(blockCh)
    blockCh  <- common.QueryBlockResult{Block: rawBlock, Chain: blockchain.Bitcoin}
  }(mqdata)
  createBlockResul := <- sqldb.CreateBitcoinBlockWithUTXOs(blockCh)
  if createBlockResul.Error != nil{
    configure.Sugar.Fatal(createBlockResul.Error.Error())
  }

  isTracking := true
  trackHeight := mqdata.Height - 1
  for isTracking {
    ch := b.Query.Block(trackHeight)
    isTracking, trackHeight = sqldb.TrackBlock(mqdata.Height, isTracking, ch)
  }
}
