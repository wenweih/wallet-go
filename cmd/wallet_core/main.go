package main

import (
	"flag"
	"net"
	"net/url"
	"strings"
	"google.golang.org/grpc"
	pb "wallet-transition/pkg/pb"
	"wallet-transition/pkg/rpc"
	"wallet-transition/pkg/blockchain"
	"wallet-transition/pkg/configure"
	"google.golang.org/grpc/reflection"
)

func main() {
	// TODO: remove bitcoin mode flag, it should be determined by rpc client
	var bitcoinmode string
	flag.StringVar(&bitcoinmode, "bitcoinmode", "mainnet", "btc base chain mode: testnet, regtest or mainnet")
	flag.Parse()
	bitcoinNet, err := blockchain.BitcoinNet(bitcoinmode)
	if err != nil {
		configure.Sugar.Fatal(err.Error())
	}

	u, err := url.Parse(strings.Join([]string{"//", configure.Config.WalletCoreRPCURL}, ""))
	if err != nil {
		configure.Sugar.Info("Parse WalletRPCURL error: ", err.Error())
	}
	_, port, err := net.SplitHostPort(u.Host)
	if err != nil {
		configure.Sugar.Fatal("rpc server error: ", err.Error())
	}

	lis, err := net.Listen("tcp", strings.Join([]string{":", port}, ""))
	if err != nil {
		configure.Sugar.Fatal("failed to listen: %v", err)
	}

	rpcServer := grpc.NewServer()
	pb.RegisterWalletCoreServer(rpcServer, &rpc.WalletCoreServerRPC{BTCNet: bitcoinNet})
	reflection.Register(rpcServer)
	if err := rpcServer.Serve(lis); err != nil {
		configure.Sugar.Info("failed to serve: ", err.Error())
	}
}
