package middlewares

import (
	"fmt"
	"net"
	"sync"
	"time"

	"greenlight/pkg/httphelpers"
	"greenlight/pkg/jsonlog"

	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"
)

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
