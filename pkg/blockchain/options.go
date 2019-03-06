package blockchain
// https://github.com/tmrts/go-patterns/blob/master/idiom/functional-options.md

// ChainID chainid option
func ChainID(chainID string) ChainsOption {
  return func(args *ChainsOptions)  {
    args.ChainID = chainID
  }
}

// ChainFrom from option
func ChainFrom(from string) ChainsOption {
  return func(args *ChainsOptions)  {
    args.From = from
  }
}

// ChainVinAmount vin amount option
func ChainVinAmount(amount int64) ChainsOption {
  return func(args *ChainsOptions)  {
    args.VinAmount = amount
  }
}

// ModeBTC btc mode option
// func ModeBTC(mode string) ChainsOption {
//   return func(args *ChainsOptions)  {
//     args.ModeBTC = mode
//   }
// }

// NewChainsOptions chainsoptions constructor
func NewChainsOptions(setters ...ChainsOption) *ChainsOptions {
  args := &ChainsOptions{}

  for _, setter := range setters {
    setter(args)
  }
  return args
}
