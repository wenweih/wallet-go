package rpc

import (
  "math/big"
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
  "github.com/btcsuite/btcutil/hdkeychain"
  "github.com/ethereum/go-ethereum/crypto"
  "github.com/ethereum/go-ethereum/core/types"
  "github.com/btcsuite/btcd/chaincfg"
)

// WalletCoreServerRPC WalletCore rpc server
type WalletCoreServerRPC struct {}

// Address walletcore server: address method
func (s *WalletCoreServerRPC) Address(ctx context.Context, in *proto.AddressReq) (*proto.AddressResp, error) {
  var address string
  switch in.Asset {
  case "btc":
    add, err := genBTCAddress()
    if err != nil {
      return nil, err
    }
    address = add.EncodeAddress()
  case "eth":
    add, err := genETHAddress()
    if err != nil {
      return nil ,err
    }
    address = *add
  default:
  }
  return &proto.AddressResp{Address: address}, nil
}

// SignTx sign raw tx
func (s *WalletCoreServerRPC) SignTx(ctx context.Context, in *proto.SignTxReq) (*proto.SignTxResp, error) {
  ldb, err := db.NewLDB(in.Asset)
  if err != nil {
    return nil, err
  }
  defer ldb.Close()
  from := in.From
  if in.Asset == "eth" {
    from = strings.ToLower(from)
  }
  priv, err := ldb.Get([]byte(from), nil)
  if err != nil && strings.Contains(err.Error(), "leveldb: not found") {
    return nil, errors.New(strings.Join([]string{"Address:", in.From, " not found: ", err.Error()}, ""))
  }
  switch in.Asset {
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
    fromAddress, _ := btcutil.DecodeAddress(in.From, &chaincfg.RegressionNetParams)
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
  case "eth":
    tx, err := blockchain.DecodeETHTx(in.HexUnsignedTx)
    if err != nil {
      return nil, errors.New(strings.Join([]string{"fail to DecodeETHTx", err.Error()}, ":"))
    }

    ecPriv, err := crypto.ToECDSA(priv)
    if err != nil {
      return nil, errors.New(strings.Join([]string{"Get private key error: ", err.Error()}, " "))
    }

    chainID, _ := new(big.Int).SetString(in.Network, 10)
    signtx, err := types.SignTx(tx, types.NewEIP155Signer(chainID), ecPriv)
    if err != nil {
      return nil, errors.New(strings.Join([]string{"sign tx error", err.Error()}, " "))
    }
    txHex, err := blockchain.EncodeETHTx(signtx)
    return &proto.SignTxResp{Result: true, HexSignedTx: *txHex}, nil
  }
  return nil, nil
}

func genBTCAddress() (*btcutil.AddressPubKeyHash, error) {
  ldb, err := db.NewLDB("btc")
  if err != nil {
    return nil, err
  }

  seed, err := hdkeychain.GenerateSeed(hdkeychain.RecommendedSeedLen)
  if err != nil {
    return nil, errors.New(strings.Join([]string{"GenerateSeed err", err.Error()}, ":"))
  }

  key, err := hdkeychain.NewMaster(seed, &chaincfg.RegressionNetParams)
  if err != nil {
    return nil, errors.New(strings.Join([]string{"NewMaster err", err.Error()}, ":"))
  }
  add, err := key.Address(&chaincfg.RegressionNetParams)
  if err != nil {
    return nil, errors.New(strings.Join([]string{"NewAddressPubKeyHash err", err.Error()}, ":"))
  }

  _, err = ldb.Get([]byte(add.EncodeAddress()), nil)
  if err != nil && strings.Contains(err.Error(), "leveldb: not found") && key.IsPrivate(){
    priv, err := key.ECPrivKey()
    if err != nil {
      return nil, errors.New(strings.Join([]string{"master key to ec privite key error:", err.Error()}, ""))
    }

    wif, err := btcutil.NewWIF(priv, &chaincfg.RegressionNetParams, true)
    if err != nil {
      return nil, errors.New(strings.Join([]string{"btcec priv to wif:", err.Error()}, ""))
    }
    if err := ldb.Put([]byte(add.EncodeAddress()), []byte(wif.String()), nil); err != nil {
      return nil, errors.New(strings.Join([]string{"put privite key to leveldb error:", err.Error()}, ""))
    }
  }else if err != nil {
    return nil, errors.New(strings.Join([]string{"Fail to add address:", add.EncodeAddress(), " ", err.Error()}, ""))
  }
  ldb.Close()

  return add, nil
}


func genETHAddress() (*string, error) {
  ldb, err := db.NewLDB("eth")
  if err != nil {
    return nil, err
  }
  privateKey, err := crypto.GenerateKey()
  if err != nil {
    return nil, errors.New(strings.Join([]string{"fail to generate ethereum key", err.Error()}, ":"))
  }
  privateKeyBytes := crypto.FromECDSA(privateKey)
  address := crypto.PubkeyToAddress(privateKey.PublicKey).Hex()

  _, err = ldb.Get([]byte(strings.ToLower(address)), nil)
  if err != nil && strings.Contains(err.Error(), "leveldb: not found") {
    if err := ldb.Put([]byte(strings.ToLower(address)), privateKeyBytes, nil); err != nil {
      return nil, errors.New(strings.Join([]string{"put privite key to leveldb error:", err.Error()}, ""))
    }
  }else if err != nil {
    return nil, errors.New(strings.Join([]string{"Fail to add address:", address, " ", err.Error()}, ""))
  }
  ldb.Close()
  return &address, nil
}
