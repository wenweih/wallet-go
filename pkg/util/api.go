package util

import (
  "fmt"
  "strconv"
  "strings"
  "net/http"
  "io/ioutil"
  "crypto/rsa"
  "encoding/json"
  b64 "encoding/base64"
  "github.com/gin-gonic/gin"
  "github.com/btcsuite/btcutil"
  "github.com/btcsuite/btcd/txscript"
  "github.com/btcsuite/btcd/chaincfg"
  "wallet-transition/pkg/configure"
  "wallet-transition/pkg/db"
)

// GinEngine api engine
func GinEngine() *gin.Engine {
  gin.SetMode(gin.ReleaseMode)
  r := gin.New()
  r.Use(gin.Logger())
  r.Use(gin.Recovery())

  privBytes, err := ioutil.ReadFile(strings.Join([]string{configure.HomeDir(), "wallet_priv.pem"}, "/"))
  if err != nil {
    configure.Sugar.Fatal("read priv key error: ", err.Error())
  }
  rsaPriv := BytesToPrivateKey(privBytes)
  r.Use(apiAuth(rsaPriv))
  return r
}

func noRouteMiddleware(ginInstance *gin.Engine) gin.HandlerFunc {
  return func(c *gin.Context) {
    ginInstance.NoRoute(func(c *gin.Context) {
      GinRespException(c, http.StatusNotFound, fmt.Errorf("Route Error"))
    })
  }
}

func apiAuth(rsaPriv *rsa.PrivateKey) gin.HandlerFunc {
  return func (c *gin.Context)  {
    ct := c.GetHeader("Content-Type")
    if ct != "application/json" {
      GinRespException(c, http.StatusUnauthorized, fmt.Errorf("Content-Type must be application/json"))
      return
    }
    token := c.Request.Header.Get("Authorization")
    if token == "" {
      GinRespException(c, http.StatusUnauthorized, fmt.Errorf("Authorization can't found in request header"))
      return
    }
    configure.Sugar.Info("request token: ", token)
    decodeToken, err := b64.StdEncoding.DecodeString(token)
    if err != nil {
      GinRespException(c, http.StatusForbidden, fmt.Errorf("Decode Token error"))
      return
    }

    decryptoParamBytes, err := DecryptWithPrivateKey(decodeToken, rsaPriv)
    if err != nil {
      GinRespException(c, http.StatusForbidden, fmt.Errorf("Decrypt Token %s", err))
      return
    }

    var params AddressParams
    if err = json.Unmarshal(decryptoParamBytes, &params); err != nil {
      GinRespException(c, http.StatusBadRequest, err)
      return
    }

    asset := strings.ToLower(params.Asset)
    if asset == "" {
      GinRespException(c, http.StatusBadRequest, fmt.Errorf("asset params can't be empty"))
      return
    }

    if configure.ChainAssets[asset] == "" {
      GinRespException(c, http.StatusBadRequest, fmt.Errorf("Not implement yep %s", asset))
      return
    }

    c.Set("detail", decryptoParamBytes)
    c.Set("asset", asset)
    c.Next()
  }
}

// GinRespException bad response util
func GinRespException(c *gin.Context, code int, err error) {
  c.AbortWithStatusJSON(code, &JSONAbortMsg{
    Code: code,
    Msg: err.Error(),
  })
}

// WithdrawParamsH handle withdraw endpoint request params
func WithdrawParamsH(detailParams []byte, asset string, sqldb  *db.GormDB) (*WithdrawParams, *db.SubAddress, error) {
  withdrawParams, err := transferParams(detailParams)
  if err != nil {
    return nil, nil, err
  }
  if withdrawParams.From == "" || withdrawParams.To == "" {
    return nil, nil, fmt.Errorf("From or to params can't be empty")
  }

  var (
    subAddress db.SubAddress
  )

  // query from address
  if err := sqldb.First(&subAddress, "address = ? AND asset = ?", withdrawParams.From, asset).Error; err !=nil && err.Error() == "record not found" {
    return nil, nil, fmt.Errorf("Record not found %s : %s", asset, withdrawParams.From)
  }else if err != nil {
    return nil, nil, err
  }
  return withdrawParams, &subAddress, nil
}

// WithdrawChain withdraw endpoint asset param
func WithdrawChain(chain string) string {
  omniTokenkeys := make([]string, len(configure.Config.OmniToken))
  for k := range configure.Config.OmniToken {
    omniTokenkeys = append(omniTokenkeys, k)
  }
  if Contain(chain, omniTokenkeys) {
    chain = "btc"
  }
  return chain
}

// SendToAddressParamsH send to address enpoint request params
func SendToAddressParamsH(detailParams []byte) (*WithdrawParams, error) {
  params, err := transferParams(detailParams)
  if err != nil {
    return nil, err
  }
  return params, nil
}

func transferParams(detailParams []byte) (*WithdrawParams, error) {
  var withdrawParams WithdrawParams
  if err := json.Unmarshal(detailParams, &withdrawParams); err != nil {
    return nil, err
  }

  // params
  amount, err := strconv.ParseFloat(withdrawParams.Amount, 64)
  if err != nil {
    return nil, fmt.Errorf("Fail to convert amount %s", err)
  }
  if amount <= 0 {
    return nil, fmt.Errorf("Amount can't be empty and less than 0")
  }
  return &withdrawParams, nil
}

// BTCWithdrawAddressValidate validate withdraw endpoint address params
func BTCWithdrawAddressValidate(from, to string, bitcoinnet *chaincfg.Params) ([]byte, []byte, error) {
  toAddress, err := btcutil.DecodeAddress(to, bitcoinnet)
  if err != nil {
    return nil, nil, fmt.Errorf("To address illegal %s", err)
  }

  fromAddress, err := btcutil.DecodeAddress(from, bitcoinnet)
  if err != nil {
    return nil, nil, fmt.Errorf("From address address illegal %s", err)
  }

  toPkScript, err := txscript.PayToAddrScript(toAddress)
  if err != nil {
    return nil, nil, fmt.Errorf("To address PayToAddrScript %s", err)
  }

  fromPkScript, err := txscript.PayToAddrScript(fromAddress)
  if err != nil {
    return nil, nil, fmt.Errorf("From address PayToAddrScript %s", err)
  }
  return fromPkScript, toPkScript, nil
}
