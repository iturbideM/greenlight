package router

import "github.com/gin-gonic/gin"

type Handler interface {
	CreateMovie(c *gin.Context)
	ShowMovie(c *gin.Context)
}

func InitRouter(engine *gin.RouterGroup, handler Handler) {
	movies := engine.Group("/movies")
	{
		movies.POST("", handler.CreateMovie)
		movies.GET("/:id", handler.ShowMovie)
	}
}
