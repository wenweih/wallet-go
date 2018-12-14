package util

import (
  "crypto/rand"
  "strings"
  "crypto/rsa"
  "crypto/sha512"
  "os"
  "encoding/pem"
  "crypto/x509"
  "log"
  "errors"
  "wallet-transition/pkg/configure"
)

// key, err := keystore.DecryptKey(ksBytes, configure.Config.KSPass)
//
// pubBytes, err := ioutil.ReadFile("/Users/lianxi/wallet_pub.pem")
// if err != nil {
//   configure.Sugar.Fatal(err.Error())
// }
// privBytes, err := ioutil.ReadFile("/Users/lianxi/wallet_priv.pem")
// if err != nil {
//   configure.Sugar.Fatal(err.Error())
// }
//
// sourcepriStr := hex.EncodeToString(crypto.FromECDSA(key.PrivateKey))
//
// rsaPub := util.BytesToPublicKey(pubBytes)
// encryptAccountPriv := util.EncryptWithPublicKey(crypto.FromECDSA(key.PrivateKey), rsaPub)
//
// rsaPriv := util.BytesToPrivateKey(privBytes)
// decryptAccountPriv := util.DecryptWithPrivateKey(encryptAccountPriv, rsaPriv)

// configure.Sugar.Info(strings.ToLower(key.Address.String()),
//   " source privStr: ", sourcepriStr,
//   " encryptAccountPriv: ", hex.EncodeToString(encryptAccountPriv),
//   " DecryptWithPrivateKey: ", hex.EncodeToString(decryptAccountPriv))

type pemKey string

const (
  publicKey pemKey = "pub"
  privateKey pemKey = "priv"
)

// RsaGen generate rsa util
func RsaGen(fileName string)  {
  key, err := rsa.GenerateKey(rand.Reader, 4096)
  checkError(err)
  savePEMKey(fileName, "priv", key)
  savePEMKey(fileName, "pub", key)
}

func savePEMKey(fileName string, p pemKey, key *rsa.PrivateKey) {
  pk := new(pem.Block)
  switch p {
  case "pub":
    pubASN1, err := x509.MarshalPKIXPublicKey(key.Public())
    checkError(err)
    pk = &pem.Block {
      Type:  "RSA PUBLIC KEY",
      Bytes: pubASN1,
    }
  case "priv":
    pk = &pem.Block{
  		Type:  "PRIVATE KEY",
  		Bytes: x509.MarshalPKCS1PrivateKey(key),
  	}
  default:
    log.Fatal("pemKey suport: pub or priv only")
  }

  fileNameWithType := strings.Join([]string{fileName, string(p)}, "_")
  file := strings.Join([]string{fileNameWithType, "pem"}, ".")

  keyOut, err := os.OpenFile(strings.Join([]string{configure.HomeDir(),file}, "/"),
    os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
  checkError(err)
  defer keyOut.Close()

	err = pem.Encode(keyOut, pk)
	checkError(err)
}

// BytesToPublicKey bytes to public key
func BytesToPublicKey(pub []byte) *rsa.PublicKey {
	block, _ := pem.Decode(pub)
	enc := x509.IsEncryptedPEMBlock(block)
	b := block.Bytes
	var err error
	if enc {
		b, err = x509.DecryptPEMBlock(block, nil)
    checkError(err)
	}
	ifc, err := x509.ParsePKIXPublicKey(b)
  checkError(err)
	key, ok := ifc.(*rsa.PublicKey)
	if !ok {
    checkError(errors.New("ifc to key fail"))
	}
	return key
}

// BytesToPrivateKey bytes to private key
func BytesToPrivateKey(priv []byte) *rsa.PrivateKey {
	block, _ := pem.Decode(priv)
	enc := x509.IsEncryptedPEMBlock(block)
	b := block.Bytes
	var err error
	if enc {
		b, err = x509.DecryptPEMBlock(block, nil)
    checkError(err)
	}
	key, err := x509.ParsePKCS1PrivateKey(b)
  checkError(err)
	return key
}

// EncryptWithPublicKey encrypts data with public key
func EncryptWithPublicKey(msg []byte, pub *rsa.PublicKey) []byte {
  hash := sha512.New()
  ciphertext, err := rsa.EncryptOAEP(hash, rand.Reader, pub, msg, nil)
	if err != nil {
    checkError(err)
	}
  return ciphertext
}

// DecryptWithPrivateKey decrypts data with private key
func DecryptWithPrivateKey(ciphertext []byte, priv *rsa.PrivateKey) ([]byte, error) {
	hash := sha512.New()
	plaintext, err := rsa.DecryptOAEP(hash, rand.Reader, priv, ciphertext, nil)
  if err != nil {
    return nil ,err
  }
  return plaintext, nil
}
