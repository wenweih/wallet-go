package util

import (
  "crypto/rand"
  "strings"
  "crypto/rsa"
  "os"
  "encoding/pem"
  "crypto/x509"
  "encoding/asn1"
  "log"
)

type pemKey string

const (
  publicKey pemKey = "pub"
  privateKey pemKey = "priv"
)


// RsaGen generate rsa util
func RsaGen(fileName string)  {
  key, err := rsa.GenerateKey(rand.Reader, 2048)
  checkError(err)
  savePEMKey(fileName, "pub", key)
  savePEMKey(fileName, "priv", key)
}

func savePEMKey(fileName string, p pemKey, key *rsa.PrivateKey) {
  pk := new(pem.Block)

  switch p {
  case "pub":
    pubASN1, err := asn1.Marshal(key.PublicKey)
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

  keyOut, err := os.OpenFile(strings.Join([]string{HomeDir(),file}, "/"),
    os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
  checkError(err)
  defer keyOut.Close()

	err = pem.Encode(keyOut, pk)
	checkError(err)
}
