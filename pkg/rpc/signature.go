package rpc

import (
  "fmt"
  "bytes"
  "strings"
  "context"
  "encoding/hex"
  "wallet-transition/pkg/pb"
  "wallet-transition/pkg/db"
  "github.com/btcsuite/btcutil"
  "wallet-transition/pkg/blockchain"
  "github.com/btcsuite/btcd/txscript"
)

// SignatureEOSIO eosio transaction signature
func (s *WalletCoreServerRPC) SignatureEOSIO(ctx context.Context, in *proto.SignatureEOSIOReq) (*proto.SignTxResp, error) {
  ldb, err := db.NewLDB(db.EOSLD)
  if err != nil {
    return nil, err
  }
  defer ldb.Close()

  // query from address
  priv, err := ldb.Get([]byte(in.Pubkey), nil)
  if err != nil && strings.Contains(err.Error(), "leveldb: not found") {
    return nil, fmt.Errorf("Address %s, not found %s", in.Pubkey, err)
  }

  eosChain := blockchain.EOSChain{}
  b := blockchain.NewBlockchain(nil, eosChain, nil)
  signedTx, err := b.Operator.SignedTx(in.RawTxHex, string(priv[:]), blockchain.NewChainsOptions(blockchain.ChainID(in.ChainID)))
  if err != nil {
    return nil, err
  }

  return &proto.SignTxResp{Result: true, HexSignedTx: signedTx}, nil
}

// SignatureEthereum ethereum transaction signature
func (s *WalletCoreServerRPC) SignatureEthereum(ctx context.Context, in *proto.SignatureEthereumReq) (*proto.SignTxResp, error) {
  ldb, err := db.NewLDB(db.EthereumLD)
  if err != nil {
    return nil, err
  }
  defer ldb.Close()

  // query from address
  priv, err := ldb.Get([]byte(in.Account), nil)
  if err != nil && strings.Contains(err.Error(), "leveldb: not found") {
    return nil, fmt.Errorf("Address %s, not found %s", in.Account, err)
  }

  chain := blockchain.EthereumChain{}
  b := blockchain.NewBlockchain(nil, chain, nil)
  signedTx, err := b.Operator.SignedTx(in.RawTxHex, hex.EncodeToString(priv), blockchain.NewChainsOptions(blockchain.ChainID(in.ChainID)))
  if err != nil {
    return nil, err
  }

  return &proto.SignTxResp{Result: true, HexSignedTx: signedTx}, nil
}

// SignatureBitcoincore bitcoincore transaction signature
func (s *WalletCoreServerRPC) SignatureBitcoincore(ctx context.Context, in *proto.SignatureBitcoincoreReq) (*proto.SignTxResp, error) {
  ldb, err := db.NewLDB(db.BitcoinCoreLD)
  if err != nil {
    return nil, err
  }
  defer ldb.Close()

  // query from address
  priv, err := ldb.Get([]byte(in.From), nil)
  if err != nil && strings.Contains(err.Error(), "leveldb: not found") {
    return nil, fmt.Errorf("Address: %s not found %s", in.From, err)
  }

  // https://www.experts-exchange.com/questions/29108851/How-to-correctly-create-and-sign-a-Bitcoin-raw-transaction-using-Btcutil-library.html
  tx, err := blockchain.DecodeBtcTxHex(in.RawTxHex)
  if err != nil {
    return nil, fmt.Errorf("Fail to decode raw tx %s", err)
  }

  wif, err := btcutil.DecodeWIF(string(priv[:]))
  if err != nil {
    return nil, fmt.Errorf("Fail to decode wif %s", err)
  }
  fromAddress, _ := btcutil.DecodeAddress(in.From, s.BTCNet)
  subscript, _ := txscript.PayToAddrScript(fromAddress)
  for i, txIn := range tx.MsgTx().TxIn {
    sigScript, err := txscript.SignatureScript(tx.MsgTx(), i, subscript, txscript.SigHashAll, wif.PrivKey, true)
    if err != nil {
      return nil, fmt.Errorf("SignatureScript %s", err)
    }
    txIn.SignatureScript = sigScript
  }

  //Validate signature
  flags := txscript.StandardVerifyFlags
  vm, err := txscript.NewEngine(subscript, tx.MsgTx(), 0, flags, nil, nil, in.VinAmount)
  if err != nil {
    return nil, fmt.Errorf("Txscript.NewEngine %s", err)
  }
  if err := vm.Execute(); err != nil {
    return nil, fmt.Errorf("Fail to sign tx %s", err)
  }

  // txToHex
  buf := bytes.NewBuffer(make([]byte, 0, tx.MsgTx().SerializeSize()))
  tx.MsgTx().Serialize(buf)
  txHex := hex.EncodeToString(buf.Bytes())
  return &proto.SignTxResp{Result: true, HexSignedTx: txHex}, nil
}
