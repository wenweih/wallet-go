package main

import (
  "errors"
  "strings"
  "encoding/json"
  "wallet-transition/pkg/util"
  "wallet-transition/pkg/configure"
  "github.com/btcsuite/btcutil"
)

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
