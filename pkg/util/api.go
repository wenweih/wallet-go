package util

import (
  "github.com/gin-gonic/gin"
)

// GinEngine api engine
func GinEngine() *gin.Engine {
  gin.SetMode(gin.ReleaseMode)
  r := gin.New()
  gin.New()
  r.Use(gin.Logger())
  r.Use(gin.Recovery())
  r.Use(noRouteMiddleware(r))
  return r
}

func noRouteMiddleware(ginInstance *gin.Engine) gin.HandlerFunc {
  return func(c *gin.Context) {
    ginInstance.NoRoute(func(c *gin.Context) {
      c.JSON(404, gin.H{"code": 404, "message": "Route Error"})
    })
  }
}

// GinRespException bad response util
func GinRespException(c *gin.Context, code int, err error) {
  c.JSON(code, gin.H{
    "status": code,
    "message": err.Error(),
  })
}
