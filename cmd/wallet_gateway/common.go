package main

import (
  "time"
  "strings"
  "context"
  "wallet-transition/pkg/db"
  pb "wallet-transition/pkg/pb"
)

func genAddress(asset string) (*string, error) {
  grpcClient := pb.NewWalletCoreClient(rpcConn)
  ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
  defer cancel()
  res, err := grpcClient.Address(ctx, &pb.AddressReq{Asset: asset})
  if err != nil {
    return nil, err
  }

  address := res.Address
  if asset == "eth" {
    address = strings.ToLower(address)
  }

  if err := sqldb.Create(&db.SubAddress{Address: address, Asset: asset}).Error; err != nil {
    return nil, err
  }
  return &address, nil
}
