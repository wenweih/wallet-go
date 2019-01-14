package main

import (
  "errors"
  "strings"
  "context"
  "encoding/json"
  "wallet-transition/pkg/db"
  pb "wallet-transition/pkg/pb"
  "wallet-transition/pkg/util"
  "wallet-transition/pkg/configure"
  "github.com/ethereum/go-ethereum/common"
)

func genAddress(ctx context.Context, asset string) (*string, error) {
  res, err := grpcClient.Address(ctx, &pb.AddressReq{Asset: asset})
  if err != nil {
    return nil, err
  }

  address := res.Address
  if err := sqldb.Create(&db.SubAddress{Address: address, Asset: asset}).Error; err != nil {
    return nil, err
  }
  return &address, nil
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
  }

  if !util.Contain(assetStr, keys) {
    return nil, errors.New(strings.Join([]string{assetStr, " balance query is not be supported"}, ""))
  }
  balanceParams.Asset = strings.ToLower(balanceParams.Asset)
  return &balanceParams, nil
}
