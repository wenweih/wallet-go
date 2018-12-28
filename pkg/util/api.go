package util

import (
  "errors"
  "strings"
  "net/http"
  "io/ioutil"
  "crypto/rsa"
  // "encoding/hex"
  "encoding/json"
  b64 "encoding/base64"
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
    configure.Sugar.Fatal("read priv key error: ", err.Error())
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
    urlArr := strings.Split(c.Request.RequestURI, "/")
    configure.Sugar.Info("c.Request: ", urlArr)
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
    decodeToken, err := b64.StdEncoding.DecodeString(token)
    if err != nil {
      GinRespException(c, http.StatusForbidden, errors.New("Decode Token error"))
      return
    }

   decryptoParamBytes, err := DecryptWithPrivateKey(decodeToken, rsaPriv)
    if err != nil {
      GinRespException(c, http.StatusForbidden, errors.New(strings.Join([]string{"Decrypt Token error", err.Error()}, ":")))
      return
    }

    detail, err := assetPram(decryptoParamBytes, urlArr[1])
    if err != nil {
      GinRespException(c, http.StatusInternalServerError, err)
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
    c.Set("detail", detail)
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

func assetPram(paramsByte []byte, endpoint string) (map[string]interface{}, error) {
  asset := ""
  detailParams := make(map[string]interface{})
  // var detailParams map[string]interface{}
  switch endpoint {
  case "address":
    var params AddressParams
    if err := json.Unmarshal(paramsByte, &params); err != nil {
      return nil, errors.New(strings.Join([]string{"Unmarshal AddressParams error", err.Error()}, ": "))
    }
    asset = params.Asset
  case "withdraw":
    var params WithdrawParams
    if err := json.Unmarshal(paramsByte, &params); err != nil {
      return nil, errors.New(strings.Join([]string{"Unmarshal AddressParams error", err.Error()}, ": "))
    }
    asset = params.Asset
    detailParams["from"] = params.From
    detailParams["to"] = params.To
    detailParams["amount"] = params.Amount
  }
  if asset == "" {
    return nil, errors.New("asset params can't be empty")
  }
  if !Contain(asset , configure.Config.APIASSETS) {
    return nil, errors.New(strings.Join([]string{asset, " is not supported currently, ", "only support: ", strings.Join(configure.Config.APIASSETS[:],",")}, ""))
  }
  return detailParams, nil
}

// AuthParams /address endpoint default params
type AuthParams struct {
  Asset   string                  `json:"asset"`
  Detail  map[string]interface{}  `json:"detail,omitempty"`
}

// AddressParams /address endpoint default params
type AddressParams struct {
  Asset string  `json:"asset"`
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
