package rpc

import (
  "context"
  "wallet-transition/pkg/pb"
  "wallet-transition/pkg/blockchain"
  empty "github.com/golang/protobuf/ptypes/empty"
)

// BitcoinWallet generate bitcoin wallet
func (s *WalletCoreServerRPC) BitcoinWallet(ctx context.Context, in *proto.BitcoinWalletReq) (*proto.WalletResponse, error) {
  mode, err := blockchain.BitcoinNet(in.Mode)
  if err != nil {
    return nil, err
  }
  btcChain := blockchain.BitcoinCoreChain{Mode: mode}
  b := blockchain.NewBlockchain(btcChain, nil, nil)
  address, err := b.Wallet.Create()
  if err != nil {
    return nil, err
  }
  return &proto.WalletResponse{Address: address}, nil
}

// EthereumWallet generate ethereum wallet
func (s *WalletCoreServerRPC) EthereumWallet(ctx context.Context, in *empty.Empty) (*proto.WalletResponse, error) {
  ethChain := blockchain.EthereumChain{}
  b := blockchain.NewBlockchain(ethChain, nil, nil)
  address, err := b.Wallet.Create()
  if err != nil {
    return nil, err
  }
  return &proto.WalletResponse{Address: address}, nil
}

// EOSIOWallet generate eosio key paire
func (s *WalletCoreServerRPC) EOSIOWallet(ctx context.Context, in *empty.Empty) (*proto.WalletResponse, error) {
  eosChain := blockchain.EOSChain{}
  b := blockchain.NewBlockchain(eosChain, nil, nil)
  address, err := b.Wallet.Create()
  if err != nil {
    return nil, err
  }
  return &proto.WalletResponse{Address: address}, nil
}
