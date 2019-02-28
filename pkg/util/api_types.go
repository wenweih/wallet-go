package util

// EOSIOtxParams eosio tx params
type EOSIOtxParams struct {
  Asset     string  `json:"asset"`
  Receiptor string  `json:"receiptor"`
  Amount    string  `json:"amount"`
  Memo      string  `json:"memo"`
}

// EthereumWithdrawParams ethereum/tx endpoint params
type EthereumWithdrawParams struct {
  Asset   string  `json:"asset" binding:"required"`
  From    string  `json:"from" binding:"required"`
  To      string  `json:"to" binding:"required"`
  Amount  string `json:"amount" binding:"required"`
}

// AddressParams /address endpoint default params
type AddressParams struct {
  Asset string  `json:"asset"`
}

// TxParams /tx endpoint default params
type TxParams struct {
  Asset string  `json:"asset"`
  Txid  string  `json:"txid"`
}

// WithdrawParams withdraw endpoint params
type WithdrawParams struct {
  Asset   string  `json:"asset" binding:"required"`
  From    string  `json:"from" binding:"required"`
  To      string  `json:"to" binding:"required"`
  Amount  string `json:"amount" binding:"required"`
}

// BlockParams block endpoint params
type BlockParams struct {
  Asset   string  `json:"asset" binding:"required"`
  Height  string  `json:"height" binding:"required"`
}

// BalanceParams balance endpoint params
type BalanceParams struct {
  Asset   string  `json:"asset" binding:"required"`
  Address string  `json:"address" binding:"required"`
}

// AssetWithAddress struct
type AssetWithAddress struct {
  Asset   string  `json:"asset" binding:"required"`
  Address string  `json:"address" binding:"required"`
}

// JSONAbortMsg about json
type JSONAbortMsg struct {
  Code  int `json:"code"`
  Msg   string `json:"msg"`
}
