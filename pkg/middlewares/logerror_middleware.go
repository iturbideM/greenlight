package middlewares

import (
	"fmt"

	"greenlight/pkg/jsonlog"

	"github.com/gin-gonic/gin"
)

func LogErrorMiddleware(l *jsonlog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()
		if len(c.Errors) > 0 {
			for _, err := range c.Errors {
				l.PrintError(err, nil)
			}
		} else if c.Writer.Status() >= 400 {
			l.PrintError(fmt.Errorf("HTTP %d: %s", c.Writer.Status(), c.Request.URL.Path), nil)
		}
	}
}
