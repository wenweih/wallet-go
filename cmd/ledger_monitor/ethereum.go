package main

import(
  "fmt"
  "context"
  "math/big"
  "encoding/json"
  "wallet-go/pkg/configure"
  "github.com/ethereum/go-ethereum/ethclient"
  "github.com/ethereum/go-ethereum/core/types"
)

func subHandle(orderHeight *big.Int, head *types.Header, nodeClient *ethclient.Client) (*big.Int, error) {
	ctx := context.Background()
	number := head.Number
	originBlock, err := nodeClient.BlockByNumber(ctx, head.Number)
	if err != nil {
		return nil, fmt.Errorf("Get origin block error, height: %s , %s", number.String(), err)
	}

	if orderHeight.Cmp(big.NewInt(0)) == 0 {
		orderHeight = originBlock.Number()
	}

	configure.Sugar.Info("sub message coming from ethereum,", "order height:", orderHeight.Int64(), " sub block height:", originBlock.Number().Int64())
	for blockNumber := orderHeight.Int64(); blockNumber <= originBlock.Number().Int64(); blockNumber++ {
		block, err := nodeClient.BlockByNumber(ctx, big.NewInt(blockNumber))
		if err != nil {
			configure.Sugar.Warn("Get block error, height:", blockNumber)
			continue
		}
		body, err := json.Marshal(block)
    if err != nil {
      configure.Sugar.Warn("json Marshal raw ethereum block error", err.Error())
    }
		messageClient.Publish(body, "bestblock", "fanout", "ethereum", "ethereum_best_block_queue")
		orderHeight.Add(orderHeight, big.NewInt(1))
	}
	return orderHeight, nil
}
