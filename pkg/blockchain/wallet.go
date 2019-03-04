package blockchain

import (
  "fmt"
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
    return "", fmt.Errorf("GenerateSeed %s", err)
  }

  key, err := hdkeychain.NewMaster(seed, b.Mode)
  if err != nil {
    return "", fmt.Errorf("NewMaster %s", err)
  }

	acct0, err := key.Child(hdkeychain.HardenedKeyStart + 0)
	if err != nil {
    return "", fmt.Errorf("Child 0 %s", err)
  }

	acct0Ext, err := acct0.Child(0)
	if err != nil {
    return "", fmt.Errorf("Acct0Ext %s", err)
	}

	acct0Ext10, err := acct0Ext.Child(10)
	if err != nil {
    return "", fmt.Errorf("Acct0Ext10 %s", err)
	}

	add, err := acct0Ext10.Address(b.Mode)
	if err != nil {
    return "", fmt.Errorf("Acct0Ext %s", err)
	}

  _, err = ldb.Get([]byte(add.EncodeAddress()), nil)
  if err != nil && strings.Contains(err.Error(), "leveldb: not found") && key.IsPrivate(){
		priv, err := acct0Ext10.ECPrivKey()
    if err != nil {
      return "", fmt.Errorf("Acct0Ext10 key to ec privite key %s", err)
    }

    wif, err := btcutil.NewWIF(priv, b.Mode, true)
    if err != nil {
      return "", fmt.Errorf("BTCec priv to wif %s", err)
    }
    if err := ldb.Put([]byte(add.EncodeAddress()), []byte(wif.String()), nil); err != nil {
      return "", fmt.Errorf("Save privite key to leveldb %s", err)
    }
  }else if err != nil {
    return "", fmt.Errorf("Fail to add address %s : %s", add.EncodeAddress(), err)
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
    return "", fmt.Errorf("Fail to generate ethereum key %s", err)
  }
  privateKeyBytes := crypto.FromECDSA(privateKey)
  address := strings.ToLower(crypto.PubkeyToAddress(privateKey.PublicKey).Hex())

  _, err = ldb.Get([]byte(address), nil)
  if err != nil && strings.Contains(err.Error(), "leveldb: not found") {
    if err = ldb.Put([]byte(address), privateKeyBytes, nil); err != nil {
      return "", fmt.Errorf("Save privite key to leveldb %s", err)
    }
  }else if err != nil {
    return "", fmt.Errorf("Fail to add address %s : %s", address, err)
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
    return "", fmt.Errorf("Fail to generate eos key %s", err)
  }

  wif := privateKey.String()
  pub := privateKey.PublicKey()
  _, err = ldb.Get([]byte(pub.String()), nil)
  if err != nil && strings.Contains(err.Error(), "leveldb: not found") {
    if err = ldb.Put([]byte(pub.String()), []byte(wif), nil); err != nil {
      return "", fmt.Errorf("Save privite key to leveldb %s", err)
    }
  }else if err != nil {
    return "", fmt.Errorf("Fail to add address %s : %s", pub.String(), err)
  }

  defer ldb.Close()
  return pub.String(), nil
}
