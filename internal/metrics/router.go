package metrics

import (
	"expvar"

	"github.com/gin-gonic/gin"
)

func InitRouter(engine *gin.Engine) {
	engine.GET("/debug/vars", gin.WrapH(expvar.Handler()))
}
