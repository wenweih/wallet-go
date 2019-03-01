package blockchain

import (
  "fmt"
  "strings"
  "context"
  "reflect"
  "math/big"
  "github.com/ybbus/jsonrpc"
  "wallet-transition/pkg/util"
  "wallet-transition/pkg/configure"
  "github.com/shopspring/decimal"
  "github.com/ethereum/go-ethereum"
  "github.com/ethereum/go-ethereum/common"
  "github.com/ethereum/go-ethereum/crypto"
  "github.com/ethereum/go-ethereum/core/types"
  "github.com/ethereum/go-ethereum/crypto/sha3"
)

// RawTx ethereum raw tx
func (c EthereumChain) RawTx(ctx context.Context, from, to, amount, memo, asset string) (string, error) {
  if !common.IsHexAddress(from) {
    return "", fmt.Errorf("Invalid address: %s", from)
  }
  if !common.IsHexAddress(to) {
    return "", fmt.Errorf("Invalid address: %s", from)
  }

  // const
  var (
    data []byte
    txPoolInspect *TxPoolInspect
    txPoolMaxCount uint64
  )
  gasLimit := uint64(21000) // in units
  token := configure.ChainsInfo[Ethereum].Tokens[strings.ToLower(asset)]
  etherToWei := decimal.NewFromBigInt(big.NewInt(1000000000000000000), 0)

  // transfer amount
  transferAmount, err := decimal.NewFromString(amount)
  if err != nil {
    return "", err
  }
  transferAmountDecimal := transferAmount.Mul(etherToWei)
  value, ok := new(big.Int).SetString(transferAmountDecimal.String(), 10)
  if !ok {
    return "", fmt.Errorf("Set amount error")
  }

  // account balance
  bal, err := c.Balance(ctx, from, asset, "")
  if err != nil {
    return "", err
  }
  balanceDecimal, _ := decimal.NewFromString(bal)

  // if transferAmount >= balanceAmount, return (May be ETH or token's balance)
  if balanceDecimal.LessThanOrEqual(transferAmountDecimal) {
    return "", fmt.Errorf("insufficient balance: less than or equal")
  }

  // token transfer meta: gasLimit, tx input data, value
  if token != "" && strings.ToLower(asset) != strings.ToLower(configure.ChainsInfo[Ethereum].Coin){
    tokenAddress := common.HexToAddress(token)

    transferFunSignature := []byte("transfer(address,uint256)")
    hash := sha3.NewKeccak256()
    hash.Write(transferFunSignature)
    methodID := hash.Sum(nil)[:4]
    paddedAddress := common.LeftPadBytes(common.HexToAddress(to).Bytes(), 32)
    tokenAmount, ok := new(big.Int).SetString(transferAmountDecimal.String(), 10)
    if !ok {
      return "", fmt.Errorf("Set amount error")
    }
    paddedAmount := common.LeftPadBytes(tokenAmount.Bytes(), 32)

    data = append(data, methodID...)
    data = append(data, paddedAddress...)
    data = append(data, paddedAmount...)

    gasLimit, err = c.Client.EstimateGas(ctx, ethereum.CallMsg{
      To: &tokenAddress,
      Data: data,
    })
    if err != nil {
      return "", fmt.Errorf("EstimateGas %s", err)
    }
    value = big.NewInt(0)
    to = token
  }

  // gas price
  gasPrice, err := c.Client.SuggestGasPrice(ctx)
  if err != nil {
    return "", err
  }

  // tx fee
  txFee := new(big.Int)
  txFee = txFee.Mul(gasPrice, big.NewInt(int64(gasLimit)))
  feeDecimal, _ := decimal.NewFromString(txFee.String())

  if token == "" {
    // ETH transfer
    // if totalCost > balance then return
    totalCost := transferAmountDecimal.Add(feeDecimal)
    if balanceDecimal.LessThan(totalCost) {
      totalCostBig, _ := new(big.Int).SetString(totalCost.String(), 10)
      balanceBig, _ := new(big.Int).SetString(bal, 10)
      return "", fmt.Errorf("Insufficient ETH balance %s : %s", util.ToEther(balanceBig).String(), util.ToEther(totalCostBig).String())
    }
  }else {
    // Token transfer
    ethbal, err := c.Balance(ctx, from, configure.ChainsInfo[Ethereum].Coin, "")
    if err != nil {
      return "", err
    }
    ethBalanceDecimal, _ := decimal.NewFromString(ethbal)
    if ethBalanceDecimal.LessThan(feeDecimal) {
      feeCostBig, _ := new(big.Int).SetString(feeDecimal.String(), 10)
      ethbalanceBig, _ := new(big.Int).SetString(ethbal, 10)
      return "", fmt.Errorf("Insufficient ETH balance for token transfer fee %s : %s", util.ToEther(ethbalanceBig).String(), util.ToEther(feeCostBig).String())
    }
  }

  // pendingNonceAt account
  pendingNonaceAt, err := c.Client.PendingNonceAt(ctx, common.HexToAddress(from))
  if err != nil {
    return "", err
  }

  // get real nonce in mempool
  rpcClient := jsonrpc.NewClient(configure.Config.EthRPC)
  response, err := rpcClient.Call("txpool_inspect")
  if err != nil {
    return "", err
  }
  if err = response.GetObject(&txPoolInspect); err != nil {
    return "", err
  }
  pending := reflect.ValueOf(txPoolInspect.Pending)
  if pending.Kind() == reflect.Map {
    for _, key := range pending.MapKeys() {
      address := key.Interface().(string)
      tx := reflect.ValueOf(pending.MapIndex(key).Interface())
      if tx.Kind() == reflect.Map && strings.ToLower(from) == strings.ToLower(address){
        for _, key := range tx.MapKeys() {
          count := key.Interface().(uint64)
          if count > txPoolMaxCount {
            txPoolMaxCount = count
          }
        }
      }
    }
  }
  pendingNonce := pendingNonaceAt
  if pendingNonaceAt !=0 && txPoolMaxCount + 1 > pendingNonaceAt {
    pendingNonce = txPoolMaxCount + 1
  }

  tx := types.NewTransaction(pendingNonce, common.HexToAddress(to), value, gasLimit, gasPrice, data)
  rawTxHex, err := EncodeETHTx(tx)
  if err != nil {
    return "", fmt.Errorf("Encode raw tx %s", err)
  }
  return rawTxHex, nil
}

// SignedTx ethereum tx signature
func (c EthereumChain) SignedTx(rawTxHex, wif string, options *ChainsOptions) (string, error) {
  ecPriv, err := crypto.HexToECDSA(wif)
  if err != nil {
    return "", err
  }
  tx, err := DecodeETHTx(rawTxHex)
  if err != nil {
    return "", err
  }

  chainID, _ := new(big.Int).SetString(options.ChainID, 10)
  signtx, err := types.SignTx(tx, types.NewEIP155Signer(chainID), ecPriv)
  if err != nil {
    return "", fmt.Errorf("Ethereum transaction signatrue %s", err)
  }
  txHex, err := EncodeETHTx(signtx)
  if err != nil {
    return "", err
  }
  return txHex, nil
}

// BroadcastTx ethereum tx broadcast
func (c EthereumChain) BroadcastTx(ctx context.Context, signedTxHex string) (string, error) {
  tx, err := DecodeETHTx(signedTxHex)
  if err != nil {
    return "", fmt.Errorf("Decode signed tx %s", err)
  }
  if err := c.Client.SendTransaction(ctx, tx); err != nil {
    return "", fmt.Errorf("Ethereum SendTransactionsigned %s", err)
  }
  return tx.Hash().String(), nil
}
