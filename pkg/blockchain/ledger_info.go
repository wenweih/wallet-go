package blockchain

import (
  "fmt"
  "wallet-go/pkg/common"
)

// Ledger ledger info
func (c BitcoinCoreChain) Ledger() (interface{}, error) {
  info, err := c.Client.GetBlockChainInfo()
  if err != nil {
    return nil, fmt.Errorf("Query Bitcoin ledger info error: %s ", err)
  }
  return info, nil
}

// Ledger ledger info
func (c EthereumChain) Ledger() (interface{}, error) {
  return nil, nil
}

// Ledger ledger info
func (c EOSChain) Ledger() (interface{}, error) {
  return nil, nil
}

// Block query bitcoin block interface method
func (c BitcoinCoreChain) Block(height int64) (<-chan common.QueryBlockResult) {
  blockCh := make(chan common.QueryBlockResult)
  go func (height int64)  {
    defer close(blockCh)
    blockHash, err := c.Client.GetBlockHash(height)
    if err != nil {
      blockCh <- common.QueryBlockResult{Error: fmt.Errorf("Query bitcoin block hash error: %s", err), Chain: Bitcoin}
      return
    }

    block, err := c.Client.GetBlockVerboseTxM(blockHash)
    if err != nil {
      blockCh <- common.QueryBlockResult{Error: fmt.Errorf("Query bitcoin block error %s", err), Chain: Bitcoin}
      return
    }
    blockCh <- common.QueryBlockResult{Block: block, Chain: Bitcoin}
    return
  }(height)
  return blockCh
}

// Block query ethereum block interface method
func (c EthereumChain) Block(height int64) (<-chan common.QueryBlockResult) {
  return nil
}

// Block query eos block interface method
func (c EOSChain) Block(height int64) (<-chan common.QueryBlockResult) {
  return nil
}
