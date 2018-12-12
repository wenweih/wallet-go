package main

import (
  "net/http"
  "github.com/gin-gonic/gin"
  "wallet-transition/pkg/db"
  "wallet-transition/pkg/configure"
)

func main()  {
  sqldb, err := db.NewSqlite()
  if err != nil {
    configure.Sugar.Fatal(err.Error())
  }
  defer sqldb.Close()

  gin.SetMode(gin.ReleaseMode)
  r := gin.Default()
  r.GET("/btc-best-block-notify", btcBestBlockNotifyHandle)
  if err := r.Run(":3001"); err != nil {
    configure.Sugar.Fatal(err.Error())
  }
}

func btcBestBlockNotifyHandle(c *gin.Context)  {
  c.JSON(http.StatusOK, gin.H {
    "status": http.StatusOK,
    "address": "hi",
  })
}
