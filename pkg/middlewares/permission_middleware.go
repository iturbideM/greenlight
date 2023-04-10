package middlewares

import (
	permissionsModels "greenlight/internal/permissions/models"
	"greenlight/pkg/httphelpers"

	"github.com/gin-gonic/gin"
)

type PermissionsRepo interface {
	GetAllForUser(userID int64) (permissionsModels.Permissions, error)
}

func RequireAuthenticatedUser(next gin.HandlerFunc) gin.HandlerFunc {
	return func(c *gin.Context) {
		user := httphelpers.ContextGetUser(c)
		if user.IsAnonymous() {
			httphelpers.StatusUnauthorizedResponse(c)
			c.Abort()
			return
		}

		next(c)
	}
}

func RequireActivatedUser(next gin.HandlerFunc) gin.HandlerFunc {
	fn := func(c *gin.Context) {
		user := httphelpers.ContextGetUser(c)

		if !user.Activated {
			httphelpers.StatusForbiddenResponse(c)
			c.Abort()
			return
		}

		next(c)
	}

	return RequireAuthenticatedUser(fn)
}

func RequirePermission(permissionsRepo PermissionsRepo, code string) gin.HandlerFunc {
	fn := func(c *gin.Context) {
		user := httphelpers.ContextGetUser(c)

		permissions, err := permissionsRepo.GetAllForUser(user.ID)
		if err != nil {
			httphelpers.StatusInternalServerErrorResponse(c, err)
			c.Abort()
			return
		}

		if !permissions.Include(code) {
			httphelpers.StatusForbiddenResponse(c)
			c.Abort()
			return
		}

		c.Next()
	}

	return RequireActivatedUser(fn)
}
