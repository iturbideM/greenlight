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
		// minimizar los lugares donde definimos strings magicos
		// despues tenemos un typo y no anda nada
		movies.POST("", requireWritePermission(permissionsRepo), handler.CreateMovie)
		movies.GET("", requireReadPermission(permissionsRepo), handler.ListMovies)
		movies.GET("/:id", requireReadPermission(permissionsRepo), handler.ShowMovie)
		movies.PATCH("/:id", requireWritePermission(permissionsRepo), handler.UpdateMovie)
		movies.DELETE("/:id", requireWritePermission(permissionsRepo), handler.DeleteMovie)
	}
}

func requireWritePermission(permissionsRepo PermissionsRepo) gin.HandlerFunc {
	return middlewares.RequirePermission(permissionsRepo, "movies:write")
}

func requireReadPermission(permissionsRepo PermissionsRepo) gin.HandlerFunc {
	return middlewares.RequirePermission(permissionsRepo, "movies:read")
}
