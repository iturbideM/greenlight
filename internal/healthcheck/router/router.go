package router

import "github.com/gin-gonic/gin"

type Handler interface {
	Healthcheck(c *gin.Context)
}

func InitRouter(engine *gin.RouterGroup, handler Handler) {
	engine.GET("/healthcheck", handler.Healthcheck)
}
