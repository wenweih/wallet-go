package blockchain

import (
  "bytes"
  "errors"
  "strings"
  // "io/ioutil"
  "context"
  "math/big"
  "wallet-transition/pkg/db"
  "wallet-transition/pkg/util"
  "github.com/shopspring/decimal"
  "wallet-transition/pkg/configure"
  "github.com/ethereum/go-ethereum/rlp"
  "github.com/ethereum/go-ethereum/crypto"
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
func (client *ETHRPC) GetBalanceAndPendingNonceAtAndGasPrice(ctx context.Context, address string) (*big.Int, *uint64, *big.Int, *big.Int, error) {
	balance, err := client.Client.BalanceAt(ctx, common.HexToAddress(address), nil)
	if err != nil {
		return nil, nil, nil, nil, errors.New(strings.Join([]string{"Failed to get ethereum balance from address:", address, err.Error()}, " "))
	}

	pendingNonceAt, err := client.Client.PendingNonceAt(ctx, common.HexToAddress(address))
	if err != nil {
		return nil, nil, nil, nil, errors.New(strings.Join([]string{"Failed to get account nonce from address:", address, err.Error()}, " "))
	}

	gasPrice, err := client.Client.SuggestGasPrice(ctx)
	if err != nil {
		return nil, nil, nil, nil, errors.New(strings.Join([]string{"get gasPrice error", err.Error()}, " "))
	}

  netVersion, err := client.Client.NetworkID(ctx)
  if err != nil {
    return nil, nil, nil, nil, errors.New(strings.Join([]string{"get ethereum network id error", err.Error()}, " "))
  }

	return balance, &pendingNonceAt, gasPrice, netVersion, nil
}

// SendTx send signed tx
func (client *ETHRPC) SendTx(hexSignedTx string) (*string, error){
  tx, err := DecodeETHTx(hexSignedTx)
  if err != nil {
    return nil, errors.New(strings.Join([]string{"Decode signed tx error", err.Error()}, ":"))
  }
  if err := client.Client.SendTransaction(context.Background(), tx); err != nil {
    return nil, errors.New(strings.Join([]string{"Ethereum SendTransactionsigned tx error", err.Error()}, ":"))
  }
  txid := tx.Hash().String()
  return &txid, nil
}

// RawTx ethereum raw tx
func (client *ETHRPC) RawTx(from, to string, amountF float64) (*string, *string, error){
  if !common.IsHexAddress(to) {
    err := errors.New(strings.Join([]string{"To: ", to, " isn't valid ethereum address"}, ""))
    return nil, nil, err
  }

  balance, nonce, gasPrice, netVersion, err := client.GetBalanceAndPendingNonceAtAndGasPrice(context.Background(), from)
  if err != nil {
    return nil, nil, err
  }
  chainID := netVersion.String()
  var (
    txFee = new(big.Int)
  )
  gasLimit := uint64(21000) // in units

  etherToWei := decimal.NewFromBigInt(big.NewInt(1000000000000000000), 0)
  balanceDecimal, _ := decimal.NewFromString(balance.String())
  transferAmount := decimal.NewFromFloat(amountF)
  transferAmount = transferAmount.Mul(etherToWei)
  txFee = txFee.Mul(gasPrice, big.NewInt(int64(gasLimit)))
  feeDecimal, _ := decimal.NewFromString(txFee.String())
  totalCost := transferAmount.Add(feeDecimal)
  if !totalCost.LessThanOrEqual(balanceDecimal) {
    err = errors.New(strings.Join([]string{"Account: ", from, " balance is not enough ", balanceDecimal.String(), ":", totalCost.String()}, ""))
    return nil, nil, err
  }

  amount, _ := new(big.Int).SetString(transferAmount.String(), 10)
  rawTxHex, _, err := CreateRawETHTx(*nonce, amount, gasPrice, to)
  if err != nil {
    return nil, nil, err
  }
  return &chainID, rawTxHex, nil
}

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

// EncodeETHTx encode eth tx
func EncodeETHTx(tx *types.Transaction) (*string, error) {
	txb, err := rlp.EncodeToBytes(tx)
	if err != nil {
		return nil, err
	}
	txHex := hexutil.Encode(txb)
	return &txHex, nil
}

// GenETHAddress generate ethereum account
func GenETHAddress() (*string, error) {
  ldb, err := db.NewLDB("eth")
  if err != nil {
    return nil, err
  }
  privateKey, err := crypto.GenerateKey()
  if err != nil {
    return nil, errors.New(strings.Join([]string{"fail to generate ethereum key", err.Error()}, ":"))
  }
  privateKeyBytes := crypto.FromECDSA(privateKey)
  address := crypto.PubkeyToAddress(privateKey.PublicKey).Hex()

  _, err = ldb.Get([]byte(strings.ToLower(address)), nil)
  if err != nil && strings.Contains(err.Error(), "leveldb: not found") {
    if err := ldb.Put([]byte(strings.ToLower(address)), privateKeyBytes, nil); err != nil {
      return nil, errors.New(strings.Join([]string{"put privite key to leveldb error:", err.Error()}, ""))
    }
  }else if err != nil {
    return nil, errors.New(strings.Join([]string{"Fail to add address:", address, " ", err.Error()}, ""))
  }
  ldb.Close()
  return &address, nil
}
