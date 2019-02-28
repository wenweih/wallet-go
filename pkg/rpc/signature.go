package rpc

import (
  "fmt"
  "strings"
  "context"
  "wallet-transition/pkg/pb"
  "wallet-transition/pkg/db"
  "wallet-transition/pkg/blockchain"
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
  signedTx, err := b.Operator.SignedTx(in.RawTxHex, string(priv[:]), blockchain.NewChainsOptions(blockchain.ChainID(in.ChainID)))
  if err != nil {
    return nil, err
  }

  return &proto.SignTxResp{Result: true, HexSignedTx: signedTx}, nil
}
