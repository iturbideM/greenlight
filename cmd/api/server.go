package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	healthcheckHandler "greenlight/internal/healthcheck/handlers"
	healthcheckRouter "greenlight/internal/healthcheck/router"
	moviesHandler "greenlight/internal/movies/handlers"
	moviesRouter "greenlight/internal/movies/router"
	"greenlight/pkg/httphelpers"
	"greenlight/pkg/jsonlog"
	"greenlight/pkg/middlewares"

	"github.com/gin-gonic/gin"
)

func Serve(l *jsonlog.Logger, cfg config, healtcheckHandler *healthcheckHandler.Handler, moviesHandler *moviesHandler.Handler) error {
	srv := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.port),
		Handler:      startRouter(l, cfg, healtcheckHandler, moviesHandler),
		IdleTimeout:  time.Minute,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	shutdownError := make(chan error)

	go func() {
		quit := make(chan os.Signal, 1)
		signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
		s := <-quit

		l.PrintInfo("shutting down server", map[string]string{
			"signal": s.String(),
		})

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		shutdownError <- srv.Shutdown(ctx)
	}()

	l.PrintInfo("starting server", map[string]string{
		"addr": srv.Addr,
		"env":  cfg.env,
	})

	err := srv.ListenAndServe()
	if !errors.Is(err, http.ErrServerClosed) {
		return err
	}

	err = <-shutdownError
	if err != nil {
		return err
	}

	l.PrintInfo("server stopped", map[string]string{
		"addr": srv.Addr,
	})

	return nil
}

func startRouter(l *jsonlog.Logger, cfg config, healtcheckHandler *healthcheckHandler.Handler, moviesHandler *moviesHandler.Handler) *gin.Engine {
	engine := gin.Default()
	engine.HandleMethodNotAllowed = true

	engine.NoRoute(gin.HandlerFunc(httphelpers.StatusNotFoundResponse))
	engine.NoMethod(gin.HandlerFunc(httphelpers.StatusMethodNotAllowedResponse))
	engine.Use(middlewares.LogErrorMiddleware(l))
	engine.Use(middlewares.RecoverPanic())
	engine.Use(middlewares.RateLimit(int(cfg.limiter.rps), cfg.limiter.burst, cfg.limiter.enabled, l))

	v1 := engine.Group("/v1")
	{
		healthcheckRouter.InitRouter(v1, healtcheckHandler)
		moviesRouter.InitRouter(v1, moviesHandler)
	}

	return engine
}
