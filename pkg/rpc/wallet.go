package rpc

import (
  "context"
  "wallet-transition/pkg/pb"
  "wallet-transition/pkg/blockchain"
  empty "github.com/golang/protobuf/ptypes/empty"
)

// BitcoinWallet generate bitcoin wallet
func (s *WalletCoreServerRPC) BitcoinWallet(ctx context.Context, in *empty.Empty) (*proto.WalletResponse, error) {
  btcChain := blockchain.BitcoinCoreChain{Mode: s.BTCNet}
  b := blockchain.NewBlockchain(btcChain, nil)
  address, err := b.Wallet.Create()
  if err != nil {
    return nil, err
  }
  return &proto.WalletResponse{Address: address}, nil
}

// EthereumWallet generate ethereum wallet
func (s *WalletCoreServerRPC) EthereumWallet(ctx context.Context, in *empty.Empty) (*proto.WalletResponse, error) {
  ethChain := blockchain.EthereumChain{}
  b := blockchain.NewBlockchain(ethChain, nil)
  address, err := b.Wallet.Create()
  if err != nil {
    return nil, err
  }
  return &proto.WalletResponse{Address: address}, nil
}
