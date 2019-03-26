package common

// QueryBlockResult query block result
type QueryBlockResult struct {
  Error error
  Chain string
  // Block *btcjson.GetBlockVerboseResult
  Block interface{}
}

// CreateBlockResult save block record result
type CreateBlockResult struct {
  Error error
}
