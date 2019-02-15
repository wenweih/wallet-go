package rpc

import (
  "github.com/btcsuite/btcd/chaincfg"
)

// WalletCoreServerRPC WalletCore rpc server
type WalletCoreServerRPC struct {
  BTCNet *chaincfg.Params
}
