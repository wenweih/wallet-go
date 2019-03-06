package blockchain

import (
  "fmt"
  "bytes"
  "strings"
  "context"
  "strconv"
  "encoding/hex"
  "wallet-transition/pkg/db"
  "wallet-transition/pkg/util"
  "github.com/btcsuite/btcutil"
  "wallet-transition/pkg/configure"
  "github.com/btcsuite/btcd/wire"
  "github.com/btcsuite/btcd/mempool"
  "github.com/btcsuite/btcd/txscript"
  "github.com/btcsuite/btcutil/coinset"
)

// RawTx bitcoin raw tx
func (c BitcoinCoreChain) RawTx(cxt context.Context, from, to, amount, memo, asset string) (string, error) {
  if configure.ChainAssets[asset] != Bitcoin {
    return "", fmt.Errorf("Unsupport %s in bitcoincore", asset)
  }
  amountF, err := strconv.ParseFloat(amount, 64)
  if err != nil {
    return "", err
  }
  txAmountSatoshi, err := btcutil.NewAmount(amountF)
  if err != nil {
    return "", err
  }

  fromPkScript, err := BitcoincoreAddressP2AS(from, c.Mode)
  if err != nil {
    return "", err
  }
  toPkScript, err := BitcoincoreAddressP2AS(to, c.Mode)
  if err != nil {
    return "", err
  }

  // query bitcoin chain info
  chaininfo, err := c.Client.GetBlockChainInfo()
  if err != nil {
    return "", err
  }
  feeKB, err := c.Client.EstimateFee(int64(6))
  if err != nil {
    return "", err
  }
  feeRate := mempool.SatoshiPerByte(feeKB.FeeRate)

  var (
    selectedutxos, unselectedutxos []db.UTXO
    selectedCoins coinset.Coins
  )

  // Coin Select
  if strings.ToLower(configure.ChainsInfo[Bitcoin].Coin) == strings.ToLower(asset) {
    // select coins for BTC transfer
    if selectedutxos, unselectedutxos, selectedCoins, err = CoinSelect(int64(chaininfo.Headers), txAmountSatoshi, c.Wallet.Address.UTXOs); err != nil {
      return "", fmt.Errorf("Select UTXO for tx %s", err)
    }
  }else {
    // select coins for Token transfer
    // 300: https://bitcoin.stackexchange.com/questions/1195/how-to-calculate-transaction-size-before-sending-legacy-non-segwit-p2pkh-p2sh
    inputAmount := feeRate.Fee(uint32(300))
    if selectedutxos, unselectedutxos, selectedCoins, err = CoinSelect(int64(chaininfo.Headers), inputAmount, c.Wallet.Address.UTXOs); err != nil {
      return "", fmt.Errorf("Select UTXO for tx %s", err)
    }
  }

  var vinAmount int64
  for _, coin := range selectedCoins.Coins() {
    vinAmount += int64(coin.Value())
  }
  msgTx := coinset.NewMsgTxWithInputCoins(wire.TxVersion, selectedCoins)

  token := configure.ChainsInfo[Bitcoin].Tokens[strings.ToLower(asset)]
  if token != "" && strings.ToLower(asset) != strings.ToLower(configure.ChainsInfo[Bitcoin].Coin) {
    // OmniToken transfer
    b := txscript.NewScriptBuilder()
    b.AddOp(txscript.OP_RETURN)

    omniVersion := util.Int2byte(uint64(0), 2)	// omnicore version
    txType := util.Int2byte(uint64(0), 2)	// omnicore tx type: simple send
    propertyID := configure.ChainsInfo[Bitcoin].Tokens[asset]
    tokenPropertyid, err := strconv.Atoi(propertyID)
    if err != nil {
      return "", fmt.Errorf("tokenPropertyid to int %s", err)
    }
    // tokenPropertyid := configure.Config.OmniToken["omni_first_token"].(int)
    tokenIdentifier := util.Int2byte(uint64(tokenPropertyid), 4)	// omni token identifier
    tokenAmount := util.Int2byte(uint64(txAmountSatoshi), 8)	// omni token transfer amount

    b.AddData([]byte("omni"))	// transaction maker
    b.AddData(omniVersion)
    b.AddData(txType)
    b.AddData(tokenIdentifier)
    b.AddData(tokenAmount)
    pkScript, err := b.Script()
    if err != nil {
      return "", fmt.Errorf("Bitcoin Token pkScript %s", err)
    }
    msgTx.AddTxOut(wire.NewTxOut(0, pkScript))
    txOutReference := wire.NewTxOut(0, toPkScript)
    msgTx.AddTxOut(txOutReference)
  }else {
    // BTC transfer
    txOutTo := wire.NewTxOut(int64(txAmountSatoshi), toPkScript)
    msgTx.AddTxOut(txOutTo)

    // recharge
    // 181, 34: https://bitcoin.stackexchange.com/questions/1195/how-to-calculate-transaction-size-before-sending-legacy-non-segwit-p2pkh-p2sh
    fee := feeRate.Fee(uint32(msgTx.SerializeSize() + 181 + 34))
    if (vinAmount - int64(txAmountSatoshi) - int64(fee)) > 0 {
      txOutReCharge := wire.NewTxOut((vinAmount-int64(txAmountSatoshi) - int64(fee)), fromPkScript)
      msgTx.AddTxOut(txOutReCharge)
    }else {
      selectedutxoForFee, _, selectedCoinsForFee, err := CoinSelect(int64(chaininfo.Headers), fee, unselectedutxos)
      if err != nil {
        return "", fmt.Errorf("Select UTXO for fee %s", err)
      }
      for _, coin := range selectedCoinsForFee.Coins() {
        vinAmount += int64(coin.Value())
      }
      txOutReCharge := wire.NewTxOut((vinAmount-int64(txAmountSatoshi) - int64(fee)), fromPkScript)
      msgTx.AddTxOut(txOutReCharge)
      selectedutxos = append(selectedutxos, selectedutxoForFee...)
    }
  }

  buf := bytes.NewBuffer(make([]byte, 0, msgTx.SerializeSize()))
  msgTx.Serialize(buf)
  rawTxHex := hex.EncodeToString(buf.Bytes())
  c.Wallet.SelectedUTXO = selectedutxos
  return rawTxHex, nil
}

// SignedTx bitcoin tx signature
func (c BitcoinCoreChain) SignedTx(rawTxHex, wif string, options *ChainsOptions) (string, error) {
  // https://www.experts-exchange.com/questions/29108851/How-to-correctly-create-and-sign-a-Bitcoin-raw-transaction-using-Btcutil-library.html
  tx, err := DecodeBtcTxHex(rawTxHex)
  if err != nil {
    return "", fmt.Errorf("Fail to decode raw tx %s", err)
  }

  ecPriv, err := btcutil.DecodeWIF(wif)
  if err != nil {
    return "", fmt.Errorf("Fail to decode wif %s", err)
  }
  fromAddress, _ := btcutil.DecodeAddress(options.From, c.Mode)
  subscript, _ := txscript.PayToAddrScript(fromAddress)
  for i, txIn := range tx.MsgTx().TxIn {
    sigScript, err := txscript.SignatureScript(tx.MsgTx(), i, subscript, txscript.SigHashAll, ecPriv.PrivKey, true)
    if err != nil {
      return "", fmt.Errorf("SignatureScript %s", err)
    }
    txIn.SignatureScript = sigScript
  }

  //Validate signature
  flags := txscript.StandardVerifyFlags
  vm, err := txscript.NewEngine(subscript, tx.MsgTx(), 0, flags, nil, nil, options.VinAmount)
  if err != nil {
    return "", fmt.Errorf("Txscript.NewEngine %s", err)
  }
  if err := vm.Execute(); err != nil {
    return "", fmt.Errorf("Fail to sign tx %s", err)
  }

  // txToHex
  buf := bytes.NewBuffer(make([]byte, 0, tx.MsgTx().SerializeSize()))
  tx.MsgTx().Serialize(buf)
  txHex := hex.EncodeToString(buf.Bytes())
  return txHex, nil
}

// BroadcastTx bitcoin tx broadcast
func (c BitcoinCoreChain) BroadcastTx(ctx context.Context, signedTxHex string) (string, error) {
  return "", nil
}
