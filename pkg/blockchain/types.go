package blockchain

import (
  "context"
  "wallet-transition/pkg/db"
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
  Wallet  *WalletInfo
  Client  *rpcclient.Client
}

// EthereumChain ethereum chain type
type EthereumChain struct {
  ChainID int
  Client  *ethclient.Client
}

// EOSChain EOS chain type
type EOSChain struct {
  Client  *eos.API
}

// WalletInfo wallet info
type WalletInfo struct {
  Address *db.SubAddress
  SelectedUTXO []db.UTXO
}

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
  Balance(ctx context.Context, account, symbol, code string) (string, error)
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

// BTCUTXO utxo type
type BTCUTXO struct {
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

// ChainsOptions chain info
type ChainsOptions struct {
	ChainID    string
  ModeBTC    string
}

// ChainsOption options for tx
type ChainsOption func(*ChainsOptions)
