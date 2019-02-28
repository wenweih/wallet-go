package rpc

import (
  "strings"
  "errors"
  "bytes"
  "context"
  "encoding/hex"
  "wallet-transition/pkg/db"
  "wallet-transition/pkg/pb"
  "wallet-transition/pkg/blockchain"
  "github.com/btcsuite/btcutil"
  "github.com/btcsuite/btcd/txscript"
)

// SendToAddressSignBTC btc sendtoaddress tx signature
func (s *WalletCoreServerRPC) SendToAddressSignBTC(ctx context.Context, in *proto.SendToAddressReq) (*proto.SignTxResp, error) {
  ldb, err := db.NewLDB("btc")
  if err != nil {
    return nil, err
  }
  defer ldb.Close()

  tx, err := blockchain.DecodeBtcTxHex(in.HexUnsignedTx)
  if err != nil {
    return nil, errors.New(strings.Join([]string{"fail to DecodeBtcTxHex", err.Error()}, ":"))
  }
  var subscriptNewEngine []byte
  for _, txIn := range tx.MsgTx().TxIn {
    pOutPoint := txIn.PreviousOutPoint
    for i, utxo := range in.Utxo {
      if pOutPoint.Hash.String() == utxo.Txid && pOutPoint.Index == utxo.Index {
        address, err := btcutil.DecodeAddress(utxo.Address, s.BTCNet)
        if err != nil {
          return nil, errors.New(strings.Join([]string{"DecodeAddress error", err.Error()}, ":"))
        }
        subscript, err := txscript.PayToAddrScript(address)
        if err != nil {
          return nil, errors.New(strings.Join([]string{"PayToAddrScript error", err.Error()}, ":"))
        }

        if i == 0 {
          subscriptNewEngine = subscript
        }

        priv, err := ldb.Get([]byte(utxo.Address), nil)
        if err != nil && strings.Contains(err.Error(), "leveldb: not found") {
          return nil, errors.New(strings.Join([]string{"Address:", utxo.Address, " not found: ", err.Error()}, ""))
        }

        wif, err := btcutil.DecodeWIF(string(priv[:]))
        if err != nil {
          return nil, errors.New(strings.Join([]string{"fail to decode wif", err.Error()}, ":"))
        }

        sigScript, err := txscript.SignatureScript(tx.MsgTx(), i, subscript, txscript.SigHashAll, wif.PrivKey, true)
        if err != nil {
          return nil, errors.New(strings.Join([]string{"SignatureScript error", err.Error()}, ":"))
        }
        txIn.SignatureScript = sigScript
      }
    }
  }

  //Validate signature
  flags := txscript.StandardVerifyFlags
  vm, err := txscript.NewEngine(subscriptNewEngine, tx.MsgTx(), 0, flags, nil, nil, in.VinAmount)
  if err != nil {
    return nil, errors.New(strings.Join([]string{"txscript.NewEngine: ", err.Error()}, ":"))
  }
  if err := vm.Execute(); err != nil {
    return nil, errors.New(strings.Join([]string{"fail to sign tx ", err.Error()}, ":"))
  }

  // txToHex
  buf := bytes.NewBuffer(make([]byte, 0, tx.MsgTx().SerializeSize()))
  tx.MsgTx().Serialize(buf)
  txHex := hex.EncodeToString(buf.Bytes())
  return &proto.SignTxResp{Result: true, HexSignedTx: txHex}, nil
}

// SignTx sign raw tx
func (s *WalletCoreServerRPC) SignTx(ctx context.Context, in *proto.SignTxReq) (*proto.SignTxResp, error) {
  from := in.From
  asset := in.Asset

  ldb, err := db.NewLDB(asset)
  if err != nil {
    return nil, err
  }
  defer ldb.Close()

  // query from address
  priv, err := ldb.Get([]byte(from), nil)
  if err != nil && strings.Contains(err.Error(), "leveldb: not found") {
    return nil, errors.New(strings.Join([]string{"Address:", from, " not found: ", err.Error()}, ""))
  }
  switch asset {
  case "btc":
    // https://www.experts-exchange.com/questions/29108851/How-to-correctly-create-and-sign-a-Bitcoin-raw-transaction-using-Btcutil-library.html
    tx, err := blockchain.DecodeBtcTxHex(in.HexUnsignedTx)
    if err != nil {
      return nil, errors.New(strings.Join([]string{"fail to DecodeBtcTxHex", err.Error()}, ":"))
    }

    wif, err := btcutil.DecodeWIF(string(priv[:]))
    if err != nil {
      return nil, errors.New(strings.Join([]string{"fail to decode wif", err.Error()}, ":"))
    }
    fromAddress, _ := btcutil.DecodeAddress(in.From, s.BTCNet)
    subscript, _ := txscript.PayToAddrScript(fromAddress)
    for i, txIn := range tx.MsgTx().TxIn {
      sigScript, err := txscript.SignatureScript(tx.MsgTx(), i, subscript, txscript.SigHashAll, wif.PrivKey, true)
      if err != nil {
        return nil, errors.New(strings.Join([]string{"SignatureScript error", err.Error()}, ":"))
      }
      txIn.SignatureScript = sigScript
    }

    //Validate signature
    flags := txscript.StandardVerifyFlags
    vm, err := txscript.NewEngine(subscript, tx.MsgTx(), 0, flags, nil, nil, in.VinAmount)
    if err != nil {
      return nil, errors.New(strings.Join([]string{"txscript.NewEngine: ", err.Error()}, ":"))
    }
    if err := vm.Execute(); err != nil {
      return nil, errors.New(strings.Join([]string{"fail to sign tx ", err.Error()}, ":"))
    }

    // txToHex
    buf := bytes.NewBuffer(make([]byte, 0, tx.MsgTx().SerializeSize()))
    tx.MsgTx().Serialize(buf)
    txHex := hex.EncodeToString(buf.Bytes())
    return &proto.SignTxResp{Result: true, HexSignedTx: txHex}, nil
  }
  return nil, nil
}
