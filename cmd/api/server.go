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
	permissionsRepo "greenlight/internal/permissions/repo"
	userHandler "greenlight/internal/users/handlers"
	userRepo "greenlight/internal/users/repo"
	userRouter "greenlight/internal/users/router"
	"greenlight/pkg/httphelpers"
	"greenlight/pkg/jsonlog"
	"greenlight/pkg/middlewares"
	"greenlight/pkg/taskutils"

	"github.com/gin-gonic/gin"
)

type Info struct {
	healthcheckHandler *healthcheckHandler.Handler
	moviesHandler      *moviesHandler.Handler
	userHandler        *userHandler.UserHandler
	userRepo           *userRepo.UserRepo
	permissionsRepo    *permissionsRepo.Repo
	tokenHandler       *userHandler.TokenHandler
	logger             *jsonlog.Logger
	cfg                config
}

func Serve(info Info) error {
	srv := &http.Server{
		Addr:         fmt.Sprintf(":%d", info.cfg.port),
		Handler:      startRouter(info),
		IdleTimeout:  time.Minute,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	shutdownError := make(chan error)

	go func() {
		quit := make(chan os.Signal, 1)
		signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
		s := <-quit

		info.logger.PrintInfo("shutting down server", map[string]string{
			"signal": s.String(),
		})

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		err := srv.Shutdown(ctx)
		if err != nil {
			shutdownError <- err
		}

		info.logger.PrintInfo("completing background tasks", map[string]string{
			"addr": srv.Addr,
		})

		taskutils.WaitAll()
		shutdownError <- nil
	}()

	info.logger.PrintInfo("starting server", map[string]string{
		"addr": srv.Addr,
		"env":  info.cfg.env,
	})

	err := srv.ListenAndServe()
	if !errors.Is(err, http.ErrServerClosed) {
		return err
	}

	err = <-shutdownError
	if err != nil {
		return err
	}

	info.logger.PrintInfo("server stopped", map[string]string{
		"addr": srv.Addr,
	})

	return nil
}

func startRouter(info Info) *gin.Engine {
	engine := gin.Default()
	engine.HandleMethodNotAllowed = true

	engine.NoRoute(gin.HandlerFunc(httphelpers.StatusNotFoundResponse))
	engine.NoMethod(gin.HandlerFunc(httphelpers.StatusMethodNotAllowedResponse))
	engine.Use(middlewares.LogErrorMiddleware(info.logger))
	engine.Use(middlewares.RecoverPanic())
	engine.Use(middlewares.RateLimit(int(info.cfg.limiter.rps), info.cfg.limiter.burst, info.cfg.limiter.enabled, info.logger))
	engine.Use(middlewares.Authenticate(info.userRepo))

	v1 := engine.Group("/v1")
	{
		healthcheckRouter.InitRouter(v1, info.healthcheckHandler)
		moviesRouter.InitRouter(v1, info.moviesHandler, info.permissionsRepo)
		userRouter.InitRouter(v1, info.userHandler, info.tokenHandler)
	}

	return engine
}
