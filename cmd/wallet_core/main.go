package main

import (
	"net"
	"google.golang.org/grpc"
	pb "wallet-transition/pkg/pb"
	"wallet-transition/pkg/rpc"
	"wallet-transition/pkg/configure"
	"google.golang.org/grpc/reflection"
)

func main() {
	lis, err := net.Listen("tcp", ":50051")
	if err != nil {
		configure.Sugar.Fatal("failed to listen: %v", err)
	}

	rpcServer := grpc.NewServer()
	pb.RegisterWalletCoreServer(rpcServer, &rpc.WalletCoreServerRPC{})
	reflection.Register(rpcServer)
	if err := rpcServer.Serve(lis); err != nil {
		configure.Sugar.Info("failed to serve: ", err.Error())
	}
}
