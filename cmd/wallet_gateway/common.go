package main

import (
  "errors"
  "strings"
  "context"
  "encoding/json"
  "wallet-transition/pkg/db"
  "wallet-transition/pkg/util"
  "wallet-transition/pkg/blockchain"
  "wallet-transition/pkg/configure"
  "github.com/btcsuite/btcutil"
  "github.com/ethereum/go-ethereum/common"
  pb "wallet-transition/pkg/pb"
  empty "github.com/golang/protobuf/ptypes/empty"
)

func genAddress(ctx context.Context, asset string) (string, error) {
  var address string
  c := configure.ChainAssets[asset]
  switch c {
  case blockchain.Bitcoin:
    res, err := grpcClient.BitcoinWallet(ctx, &pb.BitcoinWalletReq{Mode: bitcoinnet.Net.String()})
    if err != nil {
      return "", err
    }
    address = res.Address
  case blockchain.Ethereum:
    res, err := grpcClient.EthereumWallet(ctx, &empty.Empty{})
    if err != nil {
      return "", err
    }
    address = res.Address
  default:
    return "", errors.New(strings.Join([]string{asset, " not implement yep!"}, ""))
  }
  if err := sqldb.Create(&db.SubAddress{Address: address, Asset: asset}).Error; err != nil {
    return "", err
  }
  return address, nil
}

// chain: ethereum or omnicore
func balanceParamsH(chain, asset string, detailParams []byte) (*util.BalanceParams, error) {
  var balanceParams util.BalanceParams
  if err := json.Unmarshal(detailParams, &balanceParams); err != nil {
    return nil, err
  }

  if balanceParams.Address == "" {
    return nil, errors.New("address param is required")
  }

  assetStr := strings.ToLower(asset)

  keys := make([]string, 10)

  switch chain {
  case "ethereum":
    keys = append(keys, "eth")
    for k := range configure.Config.ETHToken {
      keys = append(keys, k)
    }

    if !common.IsHexAddress(balanceParams.Address) {
      err := errors.New(strings.Join([]string{"Address: ", balanceParams.Address, " isn't valid ethereum address"}, ""))
      return nil, err
    }
  case "omnicore":
    for k := range configure.Config.OmniToken {
      keys = append(keys, k)
    }
    _, err := btcutil.DecodeAddress(balanceParams.Address, bitcoinnet)
    if err != nil {
      e := errors.New(strings.Join([]string{"Address illegal", err.Error()}, ":"))
      return nil, e
    }
  default:
    e := errors.New("chain param illegal, only support ethereum and omnicore")
    return nil, e
  }

  if !util.Contain(assetStr, keys) {
    return nil, errors.New(strings.Join([]string{assetStr, " balance query is not be supported"}, ""))
  }
  balanceParams.Asset = strings.ToLower(balanceParams.Asset)
  return &balanceParams, nil
}
