package middlewares

import "github.com/gin-gonic/gin"

func EnableCORS() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Request.Header.Set("Access-Control-Allow-Origin", "*")

		c.Next()
	}
}
