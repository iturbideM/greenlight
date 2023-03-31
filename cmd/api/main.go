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
}

func main() {
	var cfg config

	flag.IntVar(&cfg.port, "port", 4000, "API server port")
	flag.StringVar(&cfg.env, "env", "development", "Environment (development|staging|production)")

	flag.StringVar(&cfg.db.dsn, "dsn", os.Getenv("GREENLIGHT_DB_DSN"), "PostgreSQL DSN")
	flag.IntVar(&cfg.db.maxOpenConns, "db-max-open-conns", 25, "PostgreSQL max open connections")
	flag.IntVar(&cfg.db.maxIdleConns, "db-max-idle-conns", 25, "PostgreSQL max idle connections")
	flag.StringVar(&cfg.db.maxIdleTime, "db-max-idle-time", "15m", "PostgreSQL max connection idle time")

	flag.Parse()

	logger := log.New(os.Stdout, "", log.Ldate|log.Ltime)

	db, err := openDB(cfg)
	if err != nil {
		logger.Fatal(err)
	}
	defer db.Close()
	logger.Printf("database connection pool established")

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

	v1 := engine.Group("/v1")
	{
		healthcheckRouter.InitRouter(v1, healtcheckHandler)
		moviesRouter.InitRouter(v1, moviesHandler)
	}

	srv := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.port),
		Handler:      engine,
		IdleTimeout:  time.Minute,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	logger.Printf("starting %s server on %s", cfg.env, srv.Addr)
	err = srv.ListenAndServe()
	logger.Fatal(err)
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
