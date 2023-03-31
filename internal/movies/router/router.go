package router

import "github.com/gin-gonic/gin"

type Handler interface {
	CreateMovie(c *gin.Context)
	ShowMovie(c *gin.Context)
	UpdateMovie(c *gin.Context)
	DeleteMovie(c *gin.Context)
	ListMovies(c *gin.Context)
}

func InitRouter(engine *gin.RouterGroup, handler Handler) {
	movies := engine.Group("/movies")
	{
		movies.POST("", handler.CreateMovie)
		movies.GET("", handler.ListMovies)
		movies.GET("/:id", handler.ShowMovie)
		movies.PATCH("/:id", handler.UpdateMovie)
		movies.DELETE("/:id", handler.DeleteMovie)
	}
}
