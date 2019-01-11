package main

import (
  "strings"
  "context"
  "wallet-transition/pkg/db"
  pb "wallet-transition/pkg/pb"
)

func genAddress(ctx context.Context, asset string) (*string, error) {
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
