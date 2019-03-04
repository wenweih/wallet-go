package main

import (
  "fmt"
  "strings"
  "errors"
  "math/big"
  "net/http"
  "encoding/json"
  "github.com/gin-gonic/gin"
  "wallet-transition/pkg/util"
  "wallet-transition/pkg/db"
  "wallet-transition/pkg/blockchain"
  "wallet-transition/pkg/configure"
  pb "wallet-transition/pkg/pb"
  empty "github.com/golang/protobuf/ptypes/empty"
)

func ethereumWalletHandle(c *gin.Context) {
  asset, _ := c.Get("asset")
  chain := configure.ChainAssets[asset.(string)]
  if chain == blockchain.Ethereum {
    res, err := grpcClient.EthereumWallet(c, &empty.Empty{})
    if err != nil {
      util.GinRespException(c, http.StatusInternalServerError, err)
      return
    }
    address := strings.ToLower(res.Address)
    if err := sqldb.Create(&db.SubAddress{Address: address, Asset: blockchain.Ethereum}).Error; err != nil {
      util.GinRespException(c, http.StatusInternalServerError, err)
      return
    }
    c.JSON(http.StatusOK, gin.H {
      "status": http.StatusOK,
      "address": address,
    })
  }else {
    util.GinRespException(c, http.StatusBadRequest, fmt.Errorf("%s is't ethereum asset", asset.(string)))
    return
  }
}

func ethereumBalanceHandle(c *gin.Context) {
  asset, _ := c.Get("asset")
  detailParams, _ := c.Get("detail")

  if configure.ChainAssets[asset.(string)] != blockchain.Ethereum {
    util.GinRespException(c, http.StatusBadRequest, fmt.Errorf("%s is't Ethereum asset", asset.(string)))
    return
  }

  var balanceParams util.BalanceParams
  if err := json.Unmarshal(detailParams.([]byte), &balanceParams); err != nil {
    util.GinRespException(c, http.StatusInternalServerError, err)
    return
  }

  if balanceParams.Address == "" {
    util.GinRespException(c, http.StatusInternalServerError, errors.New("address param is required"))
    return
  }

  chain := blockchain.EthereumChain{Client: ethereumClient}
  b := blockchain.NewBlockchain(nil, nil, chain)
  balance, err := b.Query.Balance(c, balanceParams.Address, balanceParams.Asset, "")
  if err != nil {
    util.GinRespException(c, http.StatusInternalServerError, err)
    return
  }
  amount, ok := new(big.Int).SetString(balance, 10)
  if !ok {
    util.GinRespException(c, http.StatusInternalServerError, errors.New("Set amount error"))
    return
  }
  c.JSON(http.StatusOK, gin.H {
    "status": http.StatusOK,
    "balance": util.ToEther(amount).String(),
  })
}

func ethereumWithdrawHandle(c *gin.Context) {
  assetParams, _ := c.Get("asset")
  detailParams, _ := c.Get("detail")

  // asset validate
  if configure.ChainAssets[assetParams.(string)] != blockchain.Ethereum {
    util.GinRespException(c, http.StatusBadRequest, fmt.Errorf("Unsupported Ethereum asset %s", assetParams.(string)))
    return
  }

  // extract params
  var params util.EthereumWithdrawParams
  if err := json.Unmarshal(detailParams.([]byte), &params); err != nil {
    util.GinRespException(c, http.StatusBadRequest, err)
    return
  }

  // sub address query by From account
  var subAddress db.SubAddress
  // query from address
  if err := sqldb.First(&subAddress, "address = ? AND asset = ?", strings.ToLower(params.From), blockchain.Ethereum).Error; err !=nil && err.Error() == "record not found" {
    util.GinRespException(c, http.StatusNotFound, fmt.Errorf("SubAddress not found in database: %s : %s", params.From, blockchain.Ethereum))
    return
  }else if err != nil {
    util.GinRespException(c, http.StatusNotFound, err)
    return
  }

  // ethereum chain
  chain := blockchain.EthereumChain{Client: ethereumClient}
  b := blockchain.NewBlockchain(nil, chain, chain)

  // raw tx
  rawTxHex, err := b.Operator.RawTx(c, params.From, params.To, params.Amount, "", params.Asset)
  if err != nil {
    util.GinRespException(c, http.StatusBadRequest, err)
    return
  }

  // query ethereum chainID
  chainID, err := chain.Client.NetworkID(c)
  if err != nil {
    util.GinRespException(c, http.StatusBadRequest, err)
    return
  }

  // ethereum tx signatrue
  res, err := grpcClient.SignatureEthereum(c, &pb.SignatureEthereumReq{Account: params.From, RawTxHex: rawTxHex, ChainID: chainID.String()})
  if err != nil {
    util.GinRespException(c, http.StatusInternalServerError, err)
    return
  }

  txid, err := b.Operator.BroadcastTx(c, res.HexSignedTx)
  if err != nil {
    util.GinRespException(c, http.StatusInternalServerError, err)
    return
  }

  c.JSON(http.StatusOK, gin.H {
    "status": http.StatusOK,
    "txid": txid,
  })

}
