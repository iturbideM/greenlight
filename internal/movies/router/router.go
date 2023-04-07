package router

import (
	"greenlight/internal/permissions/models"
	"greenlight/pkg/middlewares"

	"github.com/gin-gonic/gin"
)

type Handler interface {
	CreateMovie(c *gin.Context)
	ShowMovie(c *gin.Context)
	UpdateMovie(c *gin.Context)
	DeleteMovie(c *gin.Context)
	ListMovies(c *gin.Context)
}

type PermissionsRepo interface {
	GetAllForUser(userID int64) (models.Permissions, error)
}

func InitRouter(engine *gin.RouterGroup, handler Handler, permissionsRepo PermissionsRepo) {
	movies := engine.Group("/movies")
	{
		movies.POST("", middlewares.RequirePermission(permissionsRepo, "movies:write"), handler.CreateMovie)
		movies.GET("", middlewares.RequirePermission(permissionsRepo, "movies:read"), handler.ListMovies)
		movies.GET("/:id", middlewares.RequirePermission(permissionsRepo, "movies:read"), handler.ShowMovie)
		movies.PATCH("/:id", middlewares.RequirePermission(permissionsRepo, "movies:write"), handler.UpdateMovie)
		movies.DELETE("/:id", middlewares.RequirePermission(permissionsRepo, "movies:write"), handler.DeleteMovie)
	}
}
