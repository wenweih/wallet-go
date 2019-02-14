package blockchain

import (
  "github.com/btcsuite/btcutil"
  "github.com/btcsuite/btcd/chaincfg/chainhash"
  "github.com/btcsuite/btcd/chaincfg"
)

type chainNetwork string

const (
  // BitcoinRegTest bitcoin-core Regression test mode
  BitcoinRegTest chainNetwork = "regtest"
  // BitcoinTestNet bitcoin-core Testnet mode
  BitcoinTestNet chainNetwork = "testnet"
  // BitcoinMainnet bitcoin-core mainnet
  BitcoinMainnet chainNetwork = "mainnet"
)

// BitcoinCoreChain bitcoin-core chain type
type BitcoinCoreChain struct {
  Address string
  Mode    *chaincfg.Params
}

// EthereumChain ethereum chain type
type EthereumChain struct {
  Address string
  ChainID int
}

// TxOperator transaction operator
type TxOperator interface {
}

// ChainWallet chain wallet
type ChainWallet interface {
  Create() (string, error)
}

// Blockchain chain info
type Blockchain struct {
  Operator  TxOperator
  Wallet    ChainWallet
}

// BtcUTXO utxo type
type BtcUTXO struct {
	Txid      string  `json:"txid"`
	Amount    float64 `json:"amount"`
	Height    int64   `json:"height"`
	VoutIndex uint32  `json:"voutindex"`
}

// OmniBalance omni_getbalance response
type OmniBalance struct {
	Balance  string `json:"balance"`
	Reserved string `json:"reserved"`
	Frozen   string `json:"frozen"`
}

// SimpleCoin implements coinset Coin interface
type SimpleCoin struct {
	TxHash     *chainhash.Hash
	TxIndex    uint32
	TxValue    btcutil.Amount
	TxNumConfs int64
}

// TxPoolInspect ethereum transaction pool datatype
type TxPoolInspect struct {
  Pending map[string]map[uint64]string  `json:"pending"`
  Queued  map[string]map[uint64]string  `json:"queued"`
}
