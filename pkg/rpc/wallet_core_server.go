package rpc

import (
  "strings"
  "errors"
  "context"
  "wallet-transition/pkg/db"
  "wallet-transition/pkg/pb"
  "github.com/btcsuite/btcutil"
  "github.com/btcsuite/btcutil/hdkeychain"
  "github.com/ethereum/go-ethereum/crypto"
  "github.com/btcsuite/btcd/chaincfg"
)

// WalletCoreServerRPC WalletCore rpc server
type WalletCoreServerRPC struct {

}

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

func genBTCAddress() (*btcutil.AddressPubKeyHash, error) {
  ldb, err := db.NewLDB("btc")
  if err != nil {
    return nil, err
  }

  seed, err := hdkeychain.GenerateSeed(hdkeychain.RecommendedSeedLen)
  if err != nil {
    return nil, errors.New(strings.Join([]string{"GenerateSeed err", err.Error()}, ":"))
  }

  key, err := hdkeychain.NewMaster(seed, &chaincfg.MainNetParams)
  if err != nil {
    return nil, errors.New(strings.Join([]string{"NewMaster err", err.Error()}, ":"))
  }
  add, err := key.Address(&chaincfg.MainNetParams)
  if err != nil {
    return nil, errors.New(strings.Join([]string{"NewAddressPubKeyHash err", err.Error()}, ":"))
  }

  _, err = ldb.Get([]byte(add.EncodeAddress()), nil)
  if err != nil && strings.Contains(err.Error(), "leveldb: not found") && key.IsPrivate(){
    priv, err := key.ECPrivKey()
    if err != nil {
      return nil, errors.New(strings.Join([]string{"master key to ec privite key error:", err.Error()}, ""))
    }
    if err := ldb.Put([]byte(add.EncodeAddress()), priv.Serialize(), nil); err != nil {
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
