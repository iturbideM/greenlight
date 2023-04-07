package httphelpers

import (
	"greenlight/internal/users/models"

	"github.com/gin-gonic/gin"
)

type contextKey string

const userContextKey = contextKey("user")

func ContextSetUser(c *gin.Context, user *models.User) {
	c.Set(string(userContextKey), user)
}

func ContextGetUser(c *gin.Context) *models.User {
	user, ok := c.Get(string(userContextKey))
	if !ok {
		panic("missing user value in request context")
	}
	return user.(*models.User)
}
