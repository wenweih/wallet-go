package blockchain

import (
  "fmt"
  "encoding/hex"
  "encoding/json"
  "github.com/eoscanada/eos-go"
  "github.com/eoscanada/eos-go/token"
)

// RawTx eos raw tx
func (c EOSChain) RawTx(from, to, amount, memo, asset string) (string, error) {
  txOpts := &eos.TxOptions{}
  if err := txOpts.FillFromChain(c.Client); err != nil {
    return "", fmt.Errorf("filling tx opts: %s", err)
	}
  fromAccount := eos.AccountName(from)
  toAccount := eos.AccountName(to)
  quantity, err := eos.NewAsset(amount)
  if err != nil {
    return "", err
  }

  tx := eos.NewTransaction([]*eos.Action{token.NewTransfer(fromAccount, toAccount, quantity, memo)}, txOpts)
  txb, err := json.Marshal(tx)
  if err != nil {
    return "", err
  }
  txHex := hex.EncodeToString(txb)
  return txHex, nil
}
