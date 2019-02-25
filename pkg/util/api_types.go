package util

// EOSIOtxParams eosio tx params
type EOSIOtxParams struct {
  Asset     string  `json:"asset"`
  Receiptor string  `json:"receiptor"`
  Amount    string  `json:"amount"`
  Memo      string  `json:"memo"`
}
