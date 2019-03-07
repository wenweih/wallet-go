package main

import (
  "fmt"
  "strings"
  "strconv"
  "net/http"
  "encoding/json"
  "github.com/gin-gonic/gin"
  "wallet-transition/pkg/configure"
  "wallet-transition/pkg/blockchain"
  "wallet-transition/pkg/db"
  "wallet-transition/pkg/util"
  pb "wallet-transition/pkg/pb"
  "github.com/btcsuite/btcutil"
)

func bitcoincoreWalletHandle(c *gin.Context) {
  asset, _ := c.Get("asset")
  chain := configure.ChainAssets[asset.(string)]
  if chain == blockchain.Bitcoin {
    res, err := grpcClient.BitcoinWallet(c, &pb.BitcoinWalletReq{Mode: bitcoinnet.Net.String()})
    if err != nil {
      util.GinRespException(c, http.StatusInternalServerError, err)
      return
    }
    address := res.Address
    if err := sqldb.Create(&db.SubAddress{Address: address, Asset: blockchain.Bitcoin}).Error; err != nil {
      util.GinRespException(c, http.StatusInternalServerError, err)
      return
    }
    c.JSON(http.StatusOK, gin.H {
      "status": http.StatusOK,
      "address": address,
    })
  }else {
    util.GinRespException(c, http.StatusBadRequest, fmt.Errorf("%s is't bitcoin asset", asset.(string)))
    return
  }
}

func bitcoincoreWithdrawHandle(c *gin.Context) {
  assetParams, _ := c.Get("asset")
  detailParams, _ := c.Get("detail")

  // asset validate
  if configure.ChainAssets[assetParams.(string)] != blockchain.Bitcoin {
    util.GinRespException(c, http.StatusBadRequest, fmt.Errorf("Unsupported Bitcoincore asset %s", assetParams.(string)))
    return
  }

  // extract params
  var params util.WithdrawParams
  if err := json.Unmarshal(detailParams.([]byte), &params); err != nil {
    util.GinRespException(c, http.StatusBadRequest, err)
    return
  }

  // params
  amount, err := strconv.ParseFloat(params.Amount, 64)
  if err != nil {
    util.GinRespException(c, http.StatusBadRequest, err)
    return
  }

  txAmount, err := btcutil.NewAmount(amount)
  if err != nil {
    e := fmt.Errorf("convert utxo amount(float64) to btc amount(int64 as Satoshi) %s", err)
    util.GinRespException(c, http.StatusBadRequest, e)
    return
  }

  // sub address query by From account
  var subAddress db.SubAddress
  // query from address
  if err = sqldb.First(&subAddress, "address = ? AND asset = ?", strings.ToLower(params.From), blockchain.Bitcoin).Error; err !=nil && err.Error() == "record not found" {
    util.GinRespException(c, http.StatusNotFound, fmt.Errorf("SubAddress not found in database: %s : %s", params.From, blockchain.Bitcoin))
    return
  }else if err != nil {
    util.GinRespException(c, http.StatusNotFound, err)
    return
  }

  var b *blockchain.Blockchain

  token := configure.ChainsInfo[blockchain.Bitcoin].Tokens[strings.ToLower(params.Asset)]
  if token != "" && strings.ToLower(configure.ChainsInfo[blockchain.Bitcoin].Coin) != strings.ToLower(params.Asset){
    chain := blockchain.BitcoinCoreChain{Mode: bitcoinnet, Client: omniClient}
    b = blockchain.NewBlockchain(nil, nil, chain)
    tokenBal, err := b.Query.Balance(c, params.From, params.Asset, "")
    if err != nil {
      util.GinRespException(c, http.StatusInternalServerError, err)
      return
    }
    // params
    amountBal, err := strconv.ParseFloat(tokenBal, 64)
    if err != nil {
      util.GinRespException(c, http.StatusBadRequest, err)
      return
    }
    tokenAmount, err := btcutil.NewAmount(amountBal)
    if err != nil {
      e := fmt.Errorf("convert utxo amount(float64) to btc amount(int64 as Satoshi) %s", err)
      util.GinRespException(c, http.StatusBadRequest, e)
      return
    }

    if int64(txAmount) - int64(tokenAmount) > 0 {
      util.GinRespException(c, http.StatusBadRequest, fmt.Errorf("Insufficient balance %s : %s", tokenBal, params.Amount))
      return
    }
  }

  // query bitcoin current best height
  binfo, err := bitcoinClient.GetBlockChainInfo()
  if err != nil {
    util.GinRespException(c, http.StatusInternalServerError, err)
    return
  }
  bheader := binfo.Headers

  // query utxos, which confirmate count is more than 6
  var utxos  []db.UTXO
  confs := configure.ChainsInfo[blockchain.Bitcoin].Confirmations
	if err = sqldb.Model(subAddress).Where("height <= ? AND state = ?", bheader - int32(confs) + 1, "original").Related(&utxos).Error; err !=nil {
    util.GinRespException(c, http.StatusNotFound, err)
    return
	}
  subAddress.UTXOs = utxos

  // bitcoin chain
  chain := blockchain.BitcoinCoreChain{Mode: bitcoinnet, Client: bitcoinClient, Wallet: &blockchain.WalletInfo{Address: &subAddress}}
  bc := blockchain.NewBlockchain(nil, chain, nil)
  rawTxHex, err := bc.Operator.RawTx(c, params.From, params.To, params.Amount, "", params.Asset)
  if err != nil {
    util.GinRespException(c, http.StatusBadRequest, err)
    return
  }
  res, err := grpcClient.SignatureBitcoincore(c, &pb.SignatureBitcoincoreReq{From: params.From, RawTxHex: rawTxHex, Mode: bitcoinnet.Net.String()})
  if err != nil {
    util.GinRespException(c, http.StatusInternalServerError, err)
    return
  }

  txid, err := bc.Operator.BroadcastTx(c, res.HexSignedTx)
  if err != nil {
    util.GinRespException(c, http.StatusInternalServerError, err)
    return
  }

  for _, selectedUTXO := range chain.Wallet.SelectedUTXO {
    configure.Sugar.Info("utxo txid: ", selectedUTXO.Txid, " utxo index: ", selectedUTXO.VoutIndex)
  }

  c.JSON(http.StatusOK, gin.H {
    "status": http.StatusOK,
    "txid": txid,
  })

}
