package router

import "github.com/gin-gonic/gin"

type Handler interface {
	CreateMovies(c *gin.Context)
	ShowMovie(c *gin.Context)
}

func InitRouter(engine *gin.Engine, handler Handler) {
	engine.POST("/v1/movies", handler.CreateMovies)
	engine.GET("/v1/movies/:id", handler.ShowMovie)
}
