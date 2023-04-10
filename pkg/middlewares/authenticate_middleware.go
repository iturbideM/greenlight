package middlewares

import (
	"errors"
	"strings"

	"greenlight/internal/repositoryerrors"
	userModels "greenlight/internal/users/models"
	"greenlight/pkg/httphelpers"
	"greenlight/pkg/validator"

	"github.com/gin-gonic/gin"
)

type UserRepo interface {
	GetForToken(tokenScope, tokenPlaintext string) (*userModels.User, error)
}

func Authenticate(UserRepo UserRepo) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Vary", "Authorization")

		authorizationHeader := c.GetHeader("Authorization")
		if authorizationHeader == "" {
			httphelpers.ContextSetUser(c, userModels.AnonymousUser)
			c.Next()
			return
		}

		headerParts := strings.Split(authorizationHeader, " ")
		if len(headerParts) != 2 || strings.ToLower(headerParts[0]) != "bearer" {
			httphelpers.StatusUnauthorizedResponse(c)
			c.Abort()
			return
		}

		token := headerParts[1]

		v := validator.New()
		if userModels.ValidateTokenPlaintext(v, token); !v.Valid() {
			httphelpers.StatusUnauthorizedResponse(c)
			c.Abort()
			return
		}

		user, err := UserRepo.GetForToken(userModels.ScopeAuthentication, token)
		if err != nil {
			switch {
			case errors.Is(err, repositoryerrors.ErrRecordNotFound):
				httphelpers.StatusUnauthorizedResponse(c)
			default:
				httphelpers.StatusInternalServerErrorResponse(c, err)
			}
			c.Abort()
			return
		}

		httphelpers.ContextSetUser(c, user)
	}
}
