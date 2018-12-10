package util

import (
  "errors"
  "net/http"
  "io/ioutil"
  "crypto/rsa"
  "encoding/hex"
  "github.com/gin-gonic/gin"
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

  privBytes, err := ioutil.ReadFile("/Users/lianxi/wallet_priv.pem")
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
      GinRespException(c, http.StatusForbidden, errors.New("Decrypt Token error"))
      return
    }
    c.Set("params", decryptoParamBytes)
  }
}

// GinRespException bad response util
func GinRespException(c *gin.Context, code int, err error) {
  c.AbortWithStatusJSON(code, &JSONAbortMsg{
    Code: code,
    Msg: err.Error(),
  })
}

// AddressParams /address endpoint default params
type AddressParams struct {
  Asset string `json:"asset"`
}
