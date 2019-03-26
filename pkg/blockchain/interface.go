package blockchain

import (
  "context"
  "wallet-go/pkg/common"
)

// TxOperator transaction operator
type TxOperator interface {
  RawTx(ctx context.Context, from, to, amount, memo, asset string) (string, error)
  SignedTx(rawTxHex, wif string, options *ChainsOptions) (string, error)
  BroadcastTx(ctx context.Context, signedTxHex string) (string, error)
}

// ChainWallet chain wallet
type ChainWallet interface {
  Create() (string, error)
}

// ChainQuery blockchain client query
type ChainQuery interface {
  Ledger() (interface{}, error)
  Balance(ctx context.Context, account, symbol, code string) (string, error)
  Block(height int64) (<-chan common.QueryBlockResult)
}
