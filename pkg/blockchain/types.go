package blockchain

import (
  "github.com/btcsuite/btcutil"
  "github.com/btcsuite/btcd/chaincfg/chainhash"
  "github.com/btcsuite/btcd/chaincfg"
  "github.com/btcsuite/btcd/rpcclient"
  "github.com/ethereum/go-ethereum/ethclient"
  "github.com/eoscanada/eos-go"
)

const (
  // BitcoinRegTest bitcoin-core Regression test mode
  BitcoinRegTest string = "regtest"
  // BitcoinTestNet bitcoin-core Testnet mode
  BitcoinTestNet string = "testnet"
  // BitcoinMainnet bitcoin-core mainnet
  BitcoinMainnet string = "mainnet"
)

const (
  // Bitcoin bitcoin-core network
  Bitcoin string = "bitcoin"
  // Ethereum ethereum network
  Ethereum  string = "ethereum"
  // EOSIO eos network
  EOSIO    string = "eosio"
)

// BitcoinCoreChain bitcoin-core chain type
type BitcoinCoreChain struct {
  Mode    *chaincfg.Params
  Info    *WalletInfo
}

// EthereumChain ethereum chain type
type EthereumChain struct {
  ChainID int
  Info    *WalletInfo
  Client  *ethclient.Client
}

// EOSChain EOS chain type
type EOSChain struct {
  Client  *eos.API
}

// WalletInfo wallet info
type WalletInfo struct {
  Chain   string
  Address string
  Coin    string
  Tokens  map[string]interface{}
}

// TxOperator transaction operator
type TxOperator interface {
}

// ChainWallet chain wallet
type ChainWallet interface {
  Create() (string, error)
}

// ChainQuery blockchain client query
type ChainQuery interface {
  Balance(account, symbol, code string) (string, error)
}

// Blockchain chain info
type Blockchain struct {
  Operator  TxOperator
  Wallet    ChainWallet
  Query     ChainQuery
}

// BTCRPC bitcoin-core client alias
type BTCRPC struct {
	Client *rpcclient.Client
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

// ETHRPC bitcoin-core client alias
type ETHRPC struct {
	Client *ethclient.Client
}
