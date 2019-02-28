package blockchain

import (
  "fmt"
  "context"
  "encoding/hex"
  "encoding/json"
  "github.com/eoscanada/eos-go"
  "github.com/eoscanada/eos-go/token"
  "github.com/eoscanada/eos-go/ecc"
)

// RawTx eos raw tx
func (c EOSChain) RawTx(cxt context.Context, from, to, amount, memo, asset string) (string, error) {
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

// SignedTx EOSIO tx signature
func (c EOSChain) SignedTx(rawTxHex, wif string, options *ChainsOptions) (string, error) {
  txB, err := hex.DecodeString(rawTxHex)
  if err != nil {
    return "", err
  }

  var tx eos.Transaction
  if err = json.Unmarshal(txB, &tx); err != nil {
    return "", err
  }

  keyBag := eos.NewKeyBag()
  keyBag.ImportPrivateKey(wif)

  signTx := eos.NewSignedTransaction(&tx)
  privateKey, err := ecc.NewPrivateKey(wif)
  if err != nil {
    return "", err
  }
  requiredKey := privateKey.PublicKey()
  chainID, err := hex.DecodeString(options.ChainID)
  if err != nil {
    return "", err
  }

  signedTx, err := keyBag.Sign(signTx, chainID, requiredKey)
  if err != nil {
    return "", err
  }

  signedTxB, err := json.Marshal(signedTx)
  if err != nil {
    return "", err
  }
  return hex.EncodeToString(signedTxB), nil
}

// BroadcastTx EOSIO tx broadcast
func (c EOSChain) BroadcastTx(signedTxHex string) (string, error) {
  txB, err := hex.DecodeString(signedTxHex)
  if err != nil {
    return "", err
  }

  var tx eos.SignedTransaction
  if err = json.Unmarshal(txB, &tx); err != nil {
    return "", err
  }
  packedTx, err := tx.Pack(eos.CompressionNone)
  if err != nil {
    return "", err
  }
  resp, err := c.Client.PushTransaction(packedTx)
  if err != nil {
    return "", err
  }

  return resp.TransactionID, nil
}
