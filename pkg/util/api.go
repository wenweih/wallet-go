package util

import (
  "errors"
  "strings"
  "net/http"
  "io/ioutil"
  "crypto/rsa"
  "encoding/hex"
  "encoding/json"
  "github.com/gin-gonic/gin"
  "github.com/btcsuite/btcutil"
  "github.com/btcsuite/btcd/txscript"
  "github.com/btcsuite/btcd/chaincfg"
  "wallet-transition/pkg/configure"
)

// GinEngine api engine
func GinEngine() *gin.Engine {
  gin.SetMode(gin.ReleaseMode)
  r := gin.New()
  gin.New()
  r.Use(gin.Logger())
  r.Use(gin.Recovery())
  r.Use(noRouteMiddleware(r))

  privBytes, err := ioutil.ReadFile(strings.Join([]string{configure.HomeDir(), "wallet_priv.pem"}, "/"))
  if err != nil {
    configure.Sugar.Fatal(err.Error())
  }
  rsaPriv := BytesToPrivateKey(privBytes)
  r.Use(apiAuth(rsaPriv))
  return r
}

// JSONAbortMsg about json
type JSONAbortMsg struct {
  Code  int `json:"code"`
  Msg   string `json:"msg"`
}

func noRouteMiddleware(ginInstance *gin.Engine) gin.HandlerFunc {
  return func(c *gin.Context) {
    ginInstance.NoRoute(func(c *gin.Context) {
      GinRespException(c, http.StatusNotFound, errors.New("Route Error"))
    })
  }
}

func apiAuth(rsaPriv *rsa.PrivateKey) gin.HandlerFunc {
  return func (c *gin.Context)  {
    ct := c.GetHeader("Content-Type")
    if ct != "application/json" {
      GinRespException(c, http.StatusUnauthorized, errors.New("Content-Type must be application/json"))
      return
    }
    token := c.Request.Header.Get("Authorization")
    if token == "" {
      GinRespException(c, http.StatusUnauthorized, errors.New("Authorization can't found in request header"))
      return
    }
    tokenByte, err := hex.DecodeString(token)
    if err != nil {
      GinRespException(c, http.StatusForbidden, errors.New("Decode Token error"))
      return
    }

   decryptoParamBytes, err := DecryptWithPrivateKey(tokenByte, rsaPriv)
    if err != nil {
      GinRespException(c, http.StatusForbidden, errors.New(strings.Join([]string{"Decrypt Token error", err.Error()}, ":")))
      return
    }

    var params AuthParams
    if err := json.Unmarshal(decryptoParamBytes, &params); err != nil {
      GinRespException(c, http.StatusInternalServerError, errors.New("Unmarshal params error"))
      return
    }
    if params.Asset == "" {
      GinRespException(c, http.StatusBadRequest, errors.New("asset params can't be empty"))
      return
    }

    if !Contain(params.Asset , configure.Config.APIASSETS) {
      GinRespException(c, http.StatusNotFound, errors.New(strings.Join([]string{params.Asset, " is not supported currently, ", "only support: ", strings.Join(configure.Config.APIASSETS[:],",")}, "")))
      return
    }
    c.Set("detail", params.Detail)
    c.Set("asset", params.Asset)
  }
}

// GinRespException bad response util
func GinRespException(c *gin.Context, code int, err error) {
  c.AbortWithStatusJSON(code, &JSONAbortMsg{
    Code: code,
    Msg: err.Error(),
  })
}

// AuthParams /address endpoint default params
type AuthParams struct {
  Asset   string                  `json:"asset"`
  Detail  map[string]interface{}  `json:"detail,omitempty"`
}

// WithdrawParams withdraw endpoint params
type WithdrawParams struct {
  From    string  `json:"from" binding:"required"`
  To      string  `json:"to" binding:"required"`
  Amount  float64 `json:"amount" binding:"required"`
}

// BlockParams block endpoint params
type BlockParams struct {
  Height  int64  `json:"height" binding:"required"`
}

// BTCWithdrawAddressValidate validate withdraw endpoint address params
func BTCWithdrawAddressValidate(withdrawParams WithdrawParams) ([]byte, []byte, error) {
  toAddress, err := btcutil.DecodeAddress(withdrawParams.To, &chaincfg.RegressionNetParams)
  if err != nil {
    e := errors.New(strings.Join([]string{"To address illegal", err.Error()}, ":"))
    return nil, nil, e
  }

  fromAddress, err := btcutil.DecodeAddress(withdrawParams.From, &chaincfg.RegressionNetParams)
  if err != nil {
    e := errors.New(strings.Join([]string{"From address address illegal", err.Error()}, ":"))
    return nil, nil, e
  }

  toPkScript, err := txscript.PayToAddrScript(toAddress)
  if err != nil {
    e := errors.New(strings.Join([]string{"to address PayToAddrScript error", err.Error()}, ":"))
    return nil, nil, e
  }

  fromPkScript, err := txscript.PayToAddrScript(fromAddress)
  if err != nil {
    e := errors.New(strings.Join([]string{"from address PayToAddrScript error", err.Error()}, ":"))
    return nil, nil, e
  }
  return fromPkScript, toPkScript, nil
}
