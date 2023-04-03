package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"

	healthcheckHandler "greenlight/internal/healthcheck/handlers"
	healthcheckRouter "greenlight/internal/healthcheck/router"
	moviesHandler "greenlight/internal/movies/handlers"
	moviesRepo "greenlight/internal/movies/repo"
	moviesRouter "greenlight/internal/movies/router"
	"greenlight/pkg/httphelpers"
	"greenlight/pkg/jsonlog"
	"greenlight/pkg/middlewares"
)

const version = "1.0.0"

type config struct {
	port int
	env  string
	db   struct {
		dsn          string
		maxOpenConns int
		maxIdleConns int
		maxIdleTime  string
	}

	limiter struct {
		rps     float64
		burst   int
		enabled bool
	}
}

func main() {
	var cfg config

	flag.IntVar(&cfg.port, "port", 4000, "API server port")
	flag.StringVar(&cfg.env, "env", "development", "Environment (development|staging|production)")

	flag.StringVar(&cfg.db.dsn, "dsn", os.Getenv("GREENLIGHT_DB_DSN"), "PostgreSQL DSN")
	flag.IntVar(&cfg.db.maxOpenConns, "db-max-open-conns", 25, "PostgreSQL max open connections")
	flag.IntVar(&cfg.db.maxIdleConns, "db-max-idle-conns", 25, "PostgreSQL max idle connections")
	flag.StringVar(&cfg.db.maxIdleTime, "db-max-idle-time", "15m", "PostgreSQL max connection idle time")

	flag.Float64Var(&cfg.limiter.rps, "limiter-rps", 2, "Rate limiter maximum requests per second")
	flag.IntVar(&cfg.limiter.burst, "limiter-burst", 4, "Rate limiter maximum burst")
	flag.BoolVar(&cfg.limiter.enabled, "limiter-enabled", true, "Enable rate limiter")

	flag.Parse()

	logger := jsonlog.New(os.Stdout, jsonlog.LevelInfo)

	db, err := openDB(cfg)
	if err != nil {
		logger.PrintFatal(err, nil)
	}
	defer db.Close()
	logger.PrintInfo("database connection pool established", nil)

	healtcheckHandler := &healthcheckHandler.Handler{
		Version: version,
		Env:     cfg.env,
	}

	moviesHandler := &moviesHandler.Handler{
		Logger: logger,
		Repo:   moviesRepo.NewSqlxRepo(db),
	}

	engine := gin.Default()
	engine.HandleMethodNotAllowed = true

	engine.NoRoute(gin.HandlerFunc(httphelpers.StatusNotFoundResponse))
	engine.NoMethod(gin.HandlerFunc(httphelpers.StatusMethodNotAllowedResponse))
	engine.Use(middlewares.LogErrorMiddleware(logger))
	engine.Use(middlewares.RecoverPanic())
	engine.Use(middlewares.RateLimit(int(cfg.limiter.rps), cfg.limiter.burst, cfg.limiter.enabled, logger))

	v1 := engine.Group("/v1")
	{
		healthcheckRouter.InitRouter(v1, healtcheckHandler)
		moviesRouter.InitRouter(v1, moviesHandler)
	}

	srv := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.port),
		Handler:      engine,
		ErrorLog:     log.New(logger, "", 0),
		IdleTimeout:  time.Minute,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	logger.PrintInfo("starting server", map[string]string{
		"addr": srv.Addr,
		"env":  cfg.env,
	})
	err = srv.ListenAndServe()
	logger.PrintFatal(err, nil)
}

func openDB(cfg config) (*sqlx.DB, error) {
	db, err := sqlx.Connect("postgres", cfg.db.dsn)
	if err != nil {
		return nil, err
	}

	db.SetMaxOpenConns(cfg.db.maxOpenConns)
	db.SetMaxIdleConns(cfg.db.maxIdleConns)

	duration, err := time.ParseDuration(cfg.db.maxIdleTime)
	if err != nil {
		return nil, err
	}

	db.SetConnMaxIdleTime(duration)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err = db.PingContext(ctx)
	if err != nil {
		return nil, err
	}

	return db, nil
}
