package util

import (
  "crypto/rand"
  "strings"
  "crypto/rsa"
  "os"
  "encoding/pem"
  "crypto/x509"
  "log"
  "bytes"
  "errors"
  "wallet-go/pkg/configure"
)

// https://github.com/smartwalle/alipay/blob/master/encoding/rsa.go
// https://stackoverflow.com/questions/11410770/load-rsa-public-key-from-file
// https://gist.github.com/sdorra/1c95de8cb80da31610d2ad767cd6f251
// https://gist.github.com/miguelmota/3ea9286bd1d3c2a985b67cac4ba2130a
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
  fileNameWithType := strings.Join([]string{fileName, string(p)}, "_")
  var file string
  switch p {
  case "pub":
    pubASN1, err := x509.MarshalPKIXPublicKey(key.Public())
    checkError(err)
    pk = &pem.Block {
      Type:  "RSA PUBLIC KEY",
      Bytes: pubASN1,
    }
    file = strings.Join([]string{fileNameWithType, "pem"}, ".")
  case "priv":
    pk = &pem.Block{
  		Type:  "PRIVATE KEY",
  		Bytes: x509.MarshalPKCS1PrivateKey(key),
  	}
    file = strings.Join([]string{fileNameWithType, "pem"}, ".")
  default:
    log.Fatal("pemKey suport: pub or priv only")
  }

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
	ifc, err := x509.ParsePKCS1PrivateKey(b)
  checkError(err)
  // key, ok := ifc.(*rsa.PrivateKey)
  // if !ok {
  //   checkError(errors.New("ifc to key fail"))
  // }
	return ifc
}

// EncryptWithPublicKey encrypts data with public key
func EncryptWithPublicKey(msg []byte, pub *rsa.PublicKey) []byte {
  // pub.N.BitLen()/8-11
  chunks := split(msg, 117)
  var cipherData  = make([]byte, 0, 0)
  for _, d := range chunks {
		var c, e = rsa.EncryptPKCS1v15(rand.Reader, pub, d)
		if e != nil {
			checkError(e)
		}
		cipherData = append(cipherData, c...)
	}
  return cipherData

}

// DecryptWithPrivateKey decrypts data with private key
func DecryptWithPrivateKey(ciphertext []byte, priv *rsa.PrivateKey) ([]byte, error) {
  // partLen := priv.PublicKey.N.BitLen() / 8
  chunks := split(ciphertext, 128)
  buffer := bytes.NewBufferString("")

  for _, chunk := range chunks {
    decrypted, err := rsa.DecryptPKCS1v15(rand.Reader, priv, chunk)
      if err != nil {
          return nil, err
      }
      buffer.Write(decrypted)
  }
  return buffer.Bytes(), nil
}

func split(originalData []byte, packageSize int) (r [][]byte) {
	var src = make([]byte, len(originalData))
	copy(src, originalData)

	r = make([][]byte, 0)
	if len(src) <= packageSize {
		return append(r, src)
	}
	for len(src) > 0 {
		var p = src[:packageSize]
		r = append(r, p)
		src = src[packageSize:]
		if len(src) <= packageSize {
			r = append(r, src)
			break
		}
	}
	return r
}
