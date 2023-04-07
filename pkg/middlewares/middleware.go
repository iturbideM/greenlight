package middlewares

import (
	"errors"
	"fmt"
	"net"
	"strings"
	"sync"
	"time"

	"greenlight/internal/repositoryerrors"
	"greenlight/internal/users/models"
	"greenlight/pkg/httphelpers"
	"greenlight/pkg/jsonlog"
	"greenlight/pkg/validator"

	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"
)

type UserRepo interface {
	GetForToken(tokenScope, tokenPlaintext string) (*models.User, error)
}

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

func RateLimit(ratelimit, tokens int, enabled bool, l *jsonlog.Logger) gin.HandlerFunc {
	type client struct {
		limiter  *rate.Limiter
		lastSeen time.Time
	}

	var (
		mu      sync.Mutex
		clients = make(map[string]*client)
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
				clients[ip] = &client{limiter: rate.NewLimiter(rate.Limit(ratelimit), tokens)}
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

func Authenticate(UserRepo UserRepo) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Vary", "Authorization")

		authorizationHeader := c.GetHeader("Authorization")
		if authorizationHeader == "" {
			httphelpers.ContextSetUser(c, models.AnonymousUser)
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
		if models.ValidateTokenPlaintext(v, token); !v.Valid() {
			httphelpers.StatusUnauthorizedResponse(c)
			c.Abort()
			return
		}

		user, err := UserRepo.GetForToken(models.ScopeAuthentication, token)
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
