package blockchain

import (
  "fmt"
  "regexp"
  "github.com/eoscanada/eos-go"
  "github.com/ethereum/go-ethereum/rlp"
  "github.com/ethereum/go-ethereum/core/types"
  "github.com/ethereum/go-ethereum/common/hexutil"
)

var reValidAccount = regexp.MustCompile(`[a-z12345]*`)

// ToAccountNameEOS converts a eos valid name string (in) into an eos-go
// AccountName struct
func ToAccountNameEOS(in string) (out eos.AccountName, err error) {
	if !reValidAccount.MatchString(in) {
		err = fmt.Errorf("invalid characters in %q, allowed: 'a' through 'z', and '1', '2', '3', '4', '5'", in)
		return
	}

	val, _ := eos.StringToName(in)
	if eos.NameToString(val) != in {
		err = fmt.Errorf("invalid name, 13 characters maximum")
		return
	}

	if len(in) == 0 {
		err = fmt.Errorf("empty")
		return
	}
	return eos.AccountName(in), nil
}

// EncodeETHTx encode eth tx
func EncodeETHTx(tx *types.Transaction) (*string, error) {
	txb, err := rlp.EncodeToBytes(tx)
	if err != nil {
		return nil, err
	}
	txHex := hexutil.Encode(txb)
	return &txHex, nil
}
