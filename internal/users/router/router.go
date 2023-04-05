package router

import "github.com/gin-gonic/gin"

type Handler interface {
	Register(c *gin.Context)
}

func InitRouter(engine *gin.RouterGroup, handler Handler) {
	users := engine.Group("/users")
	{
		users.POST("", handler.Register)
	}
}
