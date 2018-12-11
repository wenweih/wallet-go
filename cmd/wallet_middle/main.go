package main

import (
  "net/http"
  "github.com/jinzhu/gorm"
   _ "github.com/jinzhu/gorm/dialects/sqlite"
  "github.com/gin-gonic/gin"
  "wallet-transition/pkg/configure"
)

func main()  {
  db, err := gorm.Open("sqlite3", "test.db")
  if err != nil {
    panic("failed to connect database")
  }
  defer db.Close()

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
