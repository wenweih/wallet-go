package rpc

import (
  "strings"
  "errors"
  "context"
  "wallet-transition/pkg/pb"
  "github.com/btcsuite/btcutil"
  "github.com/btcsuite/btcutil/hdkeychain"
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
    address = add.EncodeAddress()
  case "eth":
  default:

  }
  return &proto.AddressResp{Address: address}, nil
}

func genBTCAddress() (*btcutil.AddressPubKeyHash, error) {
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
  return add, nil
}
