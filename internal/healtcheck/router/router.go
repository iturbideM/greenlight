package router

import "github.com/gin-gonic/gin"

type Handler interface {
	HealthcheckHandler(c *gin.Context)
}

func InitRouter(engine *gin.Engine, handler Handler) {
	engine.GET("/v1/healthcheck", handler.HealthcheckHandler)
}
