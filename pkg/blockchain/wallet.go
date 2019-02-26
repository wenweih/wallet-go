package blockchain

import (
  "errors"
  "strings"
  "wallet-transition/pkg/db"
  "github.com/btcsuite/btcutil"
  "github.com/btcsuite/btcutil/hdkeychain"
  "github.com/ethereum/go-ethereum/crypto"
  "github.com/eoscanada/eos-go/ecc"
)

// Create generate bitcoin wallet
func (b BitcoinCoreChain) Create() (string, error) {
	ldb, err := db.NewLDB(db.BitcoinCoreLD)
  if err != nil {
    return "", err
  }

  seed, err := hdkeychain.GenerateSeed(hdkeychain.RecommendedSeedLen)
  if err != nil {
    return "", errors.New(strings.Join([]string{"GenerateSeed err", err.Error()}, ":"))
  }

  key, err := hdkeychain.NewMaster(seed, b.Mode)
  if err != nil {
    return "", errors.New(strings.Join([]string{"NewMaster err", err.Error()}, ":"))
  }

	acct0, err := key.Child(hdkeychain.HardenedKeyStart + 0)
	if err != nil {
    return "", errors.New(strings.Join([]string{"Child 0 err", err.Error()}, ":"))
  }

	acct0Ext, err := acct0.Child(0)
	if err != nil {
		return "", errors.New(strings.Join([]string{"acct0Ext err", err.Error()}, ":"))
	}

	acct0Ext10, err := acct0Ext.Child(10)
	if err != nil {
		return "", errors.New(strings.Join([]string{"acct0Ext10 err", err.Error()}, ":"))
	}

	add, err := acct0Ext10.Address(b.Mode)
	if err != nil {
		return "", errors.New(strings.Join([]string{"acct0Ext err", err.Error()}, ":"))
	}

  _, err = ldb.Get([]byte(add.EncodeAddress()), nil)
  if err != nil && strings.Contains(err.Error(), "leveldb: not found") && key.IsPrivate(){
		priv, err := acct0Ext10.ECPrivKey()
    if err != nil {
      return "", errors.New(strings.Join([]string{"acct0Ext10 key to ec privite key error:", err.Error()}, ""))
    }

    wif, err := btcutil.NewWIF(priv, b.Mode, true)
    if err != nil {
      return "", errors.New(strings.Join([]string{"btcec priv to wif:", err.Error()}, ""))
    }
    if err := ldb.Put([]byte(add.EncodeAddress()), []byte(wif.String()), nil); err != nil {
      return "", errors.New(strings.Join([]string{"put privite key to leveldb error:", err.Error()}, ""))
    }
  }else if err != nil {
    return "", errors.New(strings.Join([]string{"Fail to add address:", add.EncodeAddress(), " ", err.Error()}, ""))
  }
  defer ldb.Close()
  return add.String(), nil
}

// Create generate ethereum wallet
func (c EthereumChain) Create() (string, error) {
  ldb, err := db.NewLDB(db.EthereumLD)
  if err != nil {
    return "", err
  }
  privateKey, err := crypto.GenerateKey()
  if err != nil {
    return "", errors.New(strings.Join([]string{"fail to generate ethereum key", err.Error()}, ":"))
  }
  privateKeyBytes := crypto.FromECDSA(privateKey)
  address := strings.ToLower(crypto.PubkeyToAddress(privateKey.PublicKey).Hex())

  _, err = ldb.Get([]byte(address), nil)
  if err != nil && strings.Contains(err.Error(), "leveldb: not found") {
    if err = ldb.Put([]byte(address), privateKeyBytes, nil); err != nil {
      return "", errors.New(strings.Join([]string{"put privite key to leveldb error:", err.Error()}, ""))
    }
  }else if err != nil {
    return "", errors.New(strings.Join([]string{"Fail to add address:", address, " ", err.Error()}, ""))
  }
  defer ldb.Close()
  return address, nil
}

// Create generate eos key pair
func (c EOSChain) Create() (string, error) {
  ldb, err := db.NewLDB(db.EOSLD)
  if err != nil {
    return "", err
  }
  privateKey, err := ecc.NewRandomPrivateKey()
  if err != nil {
    return "", errors.New(strings.Join([]string{"fail to generate eos key", err.Error()}, ":"))
  }

  wif := privateKey.String()
  pub := privateKey.PublicKey()
  _, err = ldb.Get([]byte(pub.String()), nil)
  if err != nil && strings.Contains(err.Error(), "leveldb: not found") {
    if err = ldb.Put([]byte(pub.String()), []byte(wif), nil); err != nil {
      return "", errors.New(strings.Join([]string{"put privite key to leveldb error:", err.Error()}, ""))
    }
  }else if err != nil {
    return "", errors.New(strings.Join([]string{"Fail to add address:", pub.String(), " ", err.Error()}, ""))
  }

  defer ldb.Close()
  return pub.String(), nil
}
