package util

import (
  "errors"
  "strconv"
  "reflect"
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
    configure.Sugar.Info("request token: ", token)
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

    detail, asset, err := assetPram(decryptoParamBytes, urlArr[1])
    if err != nil {
      GinRespException(c, http.StatusBadRequest, err)
      return
    }

    c.Set("detail", detail)
    c.Set("asset", *asset)
  }
}

// GinRespException bad response util
func GinRespException(c *gin.Context, code int, err error) {
  c.AbortWithStatusJSON(code, &JSONAbortMsg{
    Code: code,
    Msg: err.Error(),
  })
}

func assetPram(paramsByte []byte, endpoint string) (map[string]interface{}, *string, error) {
  asset := ""
  detailParams := make(map[string]interface{})
  // var detailParams map[string]interface{}
  switch endpoint {
  case "address":
    var params AddressParams
    if err := json.Unmarshal(paramsByte, &params); err != nil {
      return nil, nil, errors.New(strings.Join([]string{"Unmarshal AddressParams error", err.Error()}, ": "))
    }
    asset = strings.ToLower(params.Asset)
  case "send", "sendtoaddress":
    var params WithdrawParams
    if err := json.Unmarshal(paramsByte, &params); err != nil {
      return nil, nil, errors.New(strings.Join([]string{"Unmarshal AddressParams error", err.Error()}, ": "))
    }
    asset = strings.ToLower(params.Asset)
    detailParams["from"] = params.From
    if asset == "eth" {
      detailParams["from"] = strings.ToLower(params.From)
    }
    detailParams["to"] = params.To
    detailParams["amount"] = params.Amount
  case "balance", "address_validator":
    var params BalanceParams
    if err := json.Unmarshal(paramsByte, &params); err != nil {
      return nil, nil, errors.New(strings.Join([]string{"Unmarshal params error", err.Error()}, ": "))
    }
    asset = strings.ToLower(params.Asset)
    detailParams["address"] = params.Address
  case "block":
    var params BlockParams
    if err := json.Unmarshal(paramsByte, &params); err != nil {
      return nil, nil, errors.New(strings.Join([]string{"Unmarshal params error", err.Error()}, ": "))
    }
    asset = strings.ToLower(params.Asset)
    detailParams["height"] = params.Height
  case "tx":
    var params TxParams
    if err := json.Unmarshal(paramsByte, &params); err != nil {
      return nil, nil, errors.New(strings.Join([]string{"Unmarshal params error", err.Error()}, ": "))
    }
    asset = strings.ToLower(params.Asset)
    detailParams["txid"] = params.Txid
  default:
    return nil, nil, errors.New("no routes")
  }
  if asset == "" {
    return nil, nil, errors.New("asset params can't be empty")
  }
  if !Contain(asset , configure.Config.APIASSETS) {
    return nil, nil, errors.New(strings.Join([]string{asset, " is not supported currently, ", "only support: ", strings.Join(configure.Config.APIASSETS[:],",")}, ""))
  }
  return detailParams, &asset, nil
}

// WithdrawParamsH handle withdraw endpoint request params
func WithdrawParamsH(detailParams interface{}, asset string, sqldb  *db.GormDB) (*WithdrawParams, *db.SubAddress, error) {
  withdrawParams, err := transferParams(detailParams)
  if err != nil {
    return nil, nil, err
  }

  if withdrawParams.From == "" || withdrawParams.To == "" {
    return nil, nil, errors.New("from or to params can't be empty")
  }

  var (
    subAddress db.SubAddress
  )

  // query from address
  if err := sqldb.First(&subAddress, "address = ? AND asset = ?", withdrawParams.From, asset).Error; err !=nil && err.Error() == "record not found" {
    return nil, nil, errors.New(strings.Join([]string{asset, " ", withdrawParams.From, " not found in database"}, ""))
  }else if err != nil {
    return nil, nil, err
  }
  return withdrawParams, &subAddress, nil
}

func SendToAddressParamsH(detailParams interface{}) (*WithdrawParams, error) {
  params, err := transferParams(detailParams)
  if err != nil {
    return nil, err
  }
  return params, nil
}

func transferParams(detailParams interface{}) (*WithdrawParams, error) {
  params := reflect.ValueOf(detailParams)
  withdrawParams := WithdrawParams{}
  if params.Kind() == reflect.Map {
    for _, key := range params.MapKeys() {
      switch key.Interface() {
      case "from":
        withdrawParams.From = params.MapIndex(key).Interface().(string)
      case "to":
        withdrawParams.To = params.MapIndex(key).Interface().(string)
      case "amount":
        withdrawParams.Amount = params.MapIndex(key).Interface().(string)
      }
    }
  }else {
    return nil, errors.New("detail params error")
  }

  // params
  amount, err := strconv.ParseFloat(withdrawParams.Amount, 64)
  if err != nil {
    return nil, errors.New(strings.Join([]string{"fail to convert amount", err.Error()}, ":"))
  }
  if amount <= 0 {
    return nil, errors.New("amount can't be empty and less than 0")
  }
  return &withdrawParams, nil
}

// AddressParams /address endpoint default params
type AddressParams struct {
  Asset string  `json:"asset"`
}

// TxParams /tx endpoint default params
type TxParams struct {
  Asset string  `json:"asset"`
  Txid  string  `json:"txid"`
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
  Asset   string  `json:"asset" binding:"required"`
  Height  string  `json:"height" binding:"required"`
}

// BalanceParams balance endpoint params
type BalanceParams struct {
  Asset   string  `json:"asset" binding:"required"`
  Address string  `json:"address" binding:"required"`
}

// AssetWithAddress struct
type AssetWithAddress struct {
  Asset   string  `json:"asset" binding:"required"`
  Address string  `json:"address" binding:"required"`
}

// BTCWithdrawAddressValidate validate withdraw endpoint address params
func BTCWithdrawAddressValidate(from, to string) ([]byte, []byte, error) {
  toAddress, err := btcutil.DecodeAddress(to, &chaincfg.RegressionNetParams)
  if err != nil {
    e := errors.New(strings.Join([]string{"To address illegal", err.Error()}, ":"))
    return nil, nil, e
  }

  fromAddress, err := btcutil.DecodeAddress(from, &chaincfg.RegressionNetParams)
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
