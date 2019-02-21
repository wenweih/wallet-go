package blockchain

import (
  "errors"
  "strings"
  "context"
  "math/big"
  "github.com/ethereum/go-ethereum/common"
  "github.com/ethereum/go-ethereum/core/types"
)

// CreateRawETHTx create eth raw tx
func CreateRawETHTx(nonce uint64, transferAmount, gasPrice *big.Int, hexAddressTo string) (*string, *string, error) {
	gasLimit := uint64(21000) // in units

	if !common.IsHexAddress(hexAddressTo) {
		return nil, nil, errors.New(strings.Join([]string{hexAddressTo, "invalidate"}, " "))
	}

	tx := types.NewTransaction(nonce, common.HexToAddress(hexAddressTo), transferAmount, gasLimit, gasPrice, nil)
	rawTxHex, err := EncodeETHTx(tx)
	if err != nil {
		return nil, nil, errors.New(strings.Join([]string{"encode raw tx error", err.Error()}, " "))
	}
	txHashHex := tx.Hash().Hex()
	return rawTxHex, &txHashHex, nil
}

// SendTx send signed tx
func (client *ETHRPC) SendTx(ctx context.Context, hexSignedTx string) (*string, error){
  tx, err := DecodeETHTx(hexSignedTx)
  if err != nil {
    return nil, errors.New(strings.Join([]string{"Decode signed tx error", err.Error()}, ":"))
  }
  if err := client.Client.SendTransaction(ctx, tx); err != nil {
    return nil, errors.New(strings.Join([]string{"Ethereum SendTransactionsigned tx error", err.Error()}, ":"))
  }
  txid := tx.Hash().String()
  return &txid, nil
}
