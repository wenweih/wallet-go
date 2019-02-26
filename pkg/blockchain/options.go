package blockchain
// https://github.com/tmrts/go-patterns/blob/master/idiom/functional-options.md

// ChainID chainid option
func ChainID(chainID string) ChainsOption {
  return func(args *ChainsOptions)  {
    args.ChainID = chainID
  }
}

// ModeBTC btc mode option
func ModeBTC(mode string) ChainsOption {
  return func(args *ChainsOptions)  {
    args.ModeBTC = mode
  }
}

// NewChainsOptions chainsoptions constructor
func NewChainsOptions(setters ...ChainsOption) *ChainsOptions {
  args := &ChainsOptions{
    ModeBTC: BitcoinMainnet,
  }

  for _, setter := range setters {
    setter(args)
  }
  return args
}
