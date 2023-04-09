package middlewares

import (
	"errors"
	"fmt"
	"net"
	"strings"
	"sync"
	"time"

	permissionsModels "greenlight/internal/permissions/models"
	"greenlight/internal/repositoryerrors"
	userModels "greenlight/internal/users/models"
	"greenlight/pkg/httphelpers"
	"greenlight/pkg/jsonlog"
	"greenlight/pkg/validator"

	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"
)

type UserRepo interface {
	GetForToken(tokenScope, tokenPlaintext string) (*userModels.User, error)
}

type PermissionsRepo interface {
	GetAllForUser(userID int64) (permissionsModels.Permissions, error)
}

// a su propio archivo
func RecoverPanic() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				httphelpers.StatusInternalServerErrorResponse(c, err.(error))
			}
		}()
		c.Next()
	}
}

// a su propio archivo
func LogErrorMiddleware(l *jsonlog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()
		if len(c.Errors) > 0 {
			for _, err := range c.Errors {
				l.PrintError(err, nil)
			}
		} else if c.Writer.Status() >= 400 {
			l.PrintError(fmt.Errorf("HTTP %d: %s", c.Writer.Status(), c.Request.URL.Path), nil)
		}
	}
}

// a su propio archivo
type rlClient struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

func RateLimit(ratelimit, tokens int, enabled bool, l *jsonlog.Logger) gin.HandlerFunc {
	var (
		mu sync.Mutex
		// hay que tener cuidado con los mapas de vida infinita, porque pueden generar un memory leak
		// una opcion seria usar un cache como redis (ideal)
		// otra opcion es usar un map con un tiempo de vida, y que se limpie cada cierto tiempo
		clients = make(map[string]*rlClient)
	)

	go func() {
		for {
			time.Sleep(time.Minute)

			mu.Lock()

			for ip, client := range clients {
				if time.Since(client.lastSeen) > 3*time.Minute {
					delete(clients, ip)
				}
			}

			mu.Unlock()
		}
	}()

	return func(c *gin.Context) {
		if enabled {
			ip, _, err := net.SplitHostPort(c.Request.RemoteAddr)
			if err != nil {
				httphelpers.StatusInternalServerErrorResponse(c, err)
				c.Abort()
				return
			}

			mu.Lock()

			if _, found := clients[ip]; !found {
				clients[ip] = &rlClient{limiter: rate.NewLimiter(rate.Limit(ratelimit), tokens)}
			}

			clients[ip].lastSeen = time.Now()

			if !clients[ip].limiter.Allow() {
				mu.Unlock()
				httphelpers.RateLimitExceededResponse(c)
				c.Abort()
				return
			}

			mu.Unlock()
		}
	}
}

// a su propio archivo
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

// a su propio archivo
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
