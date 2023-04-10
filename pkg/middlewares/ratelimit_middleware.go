package middlewares

import (
	"net"
	"sync"
	"time"

	"greenlight/pkg/httphelpers"
	"greenlight/pkg/jsonlog"

	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"
)

type rlClient struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

func RateLimit(ratelimit, tokens int, enabled bool, l *jsonlog.Logger) gin.HandlerFunc {
	var (
		mu      sync.Mutex
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
