package blockchain

import (
  "bytes"
  "errors"
  "strings"
  // "io/ioutil"
  "context"
  "math/big"
  "wallet-transition/pkg/util"
  "wallet-transition/pkg/configure"
  "github.com/ethereum/go-ethereum/rlp"
  "github.com/ethereum/go-ethereum/common"
  "github.com/ethereum/go-ethereum/ethclient"
  "github.com/ethereum/go-ethereum/core/types"
  "github.com/ethereum/go-ethereum/common/hexutil"
)

// ETHRPC bitcoin-core client alias
type ETHRPC struct {
	Client *ethclient.Client
}

// Close close rpc connect
func (client *ETHRPC) Close()  {
  client.Client.Close()
}

// NewEthClient new ethereum rpc client
func NewEthClient() (*ETHRPC, error) {
  ethClient, err := ethclient.Dial(configure.Config.EthRPCWS)
  if err != nil {
    return nil, err
  }
  return &ETHRPC{ethClient}, nil
}

var ethWalletBackupPath = strings.Join([]string{configure.Config.BackupWalletPath, "eth.backup"}, "")

// DumpETHAccount dump ethereum account from keystore
func DumpETHAccount(local bool)  {
  oldWalletServerClient, err := util.NewServerClient(configure.Config.OldETHWalletServerUser,
    configure.Config.OldETHWalletServerPass,
    configure.Config.OldETHWalletServerHost)
  if err != nil {
    configure.Sugar.Fatal(err.Error())
  }

  // pubBytes, err := ioutil.ReadFile(strings.Join([]string{configure.HomeDir(), "wallet_pub.pem"}, "/"))
  // if err != nil {
  //   configure.Sugar.Fatal(err.Error())
  // }
  // rsaPub := util.BytesToPublicKey(pubBytes)

  var ethWalletBackupPath = strings.Join([]string{configure.Config.BackupWalletPath, "eth.backup"}, "")

  if err := oldWalletServerClient.SaveEthAccount(ethWalletBackupPath); err != nil {
    configure.Sugar.Fatal(err.Error())
  }

  oldWalletServerClient.CopyRemoteFile2(ethWalletBackupPath, local)
}

// GetBalanceAndPendingNonceAtAndGasPrice construct eth tx info
func (client *ETHRPC) GetBalanceAndPendingNonceAtAndGasPrice(ctx context.Context, address string) (*big.Int, *uint64, *big.Int, error) {
	balance, err := client.Client.BalanceAt(ctx, common.HexToAddress(address), nil)
	if err != nil {
		return nil, nil, nil, errors.New(strings.Join([]string{"Failed to get ethereum balance from address:", address, err.Error()}, " "))
	}

	pendingNonceAt, err := client.Client.PendingNonceAt(ctx, common.HexToAddress(address))
	if err != nil {
		return nil, nil, nil, errors.New(strings.Join([]string{"Failed to get account nonce from address:", address, err.Error()}, " "))
	}

	gasPrice, err := client.Client.SuggestGasPrice(ctx)
	if err != nil {
		return nil, nil, nil, errors.New(strings.Join([]string{"get gasPrice error", err.Error()}, " "))
	}

	return balance, &pendingNonceAt, gasPrice, nil
}

// CreateRawETHTx create eth raw tx
func CreateRawETHTx(nonce uint64, transferAmount, gasPrice *big.Int, hexAddressFrom, hexAddressTo string) (*string, *string, error) {
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

// DecodeETHTx ethereum transaction hex
func DecodeETHTx(txHex string) (*types.Transaction, error) {
	txc, err := hexutil.Decode(txHex)
	if err != nil {
		return nil, err
	}

	var txde types.Transaction

	t, err := &txde, rlp.Decode(bytes.NewReader(txc), &txde)
	if err != nil {
		return nil, err
	}

	return t, nil
}

func EncodeETHTx(tx *types.Transaction) (*string, error) {
	txb, err := rlp.EncodeToBytes(tx)
	if err != nil {
		return nil, err
	}
	txHex := hexutil.Encode(txb)
	return &txHex, nil
}
