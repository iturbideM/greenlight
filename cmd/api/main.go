package main

import (
	"context"
	"flag"
	"os"
	"time"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"

	healthcheckHandler "greenlight/internal/healthcheck/handlers"
	moviesHandler "greenlight/internal/movies/handlers"
	moviesRepo "greenlight/internal/movies/repo"
	permissionsRepo "greenlight/internal/permissions/repo"
	userHandlers "greenlight/internal/users/handlers"
	userRepos "greenlight/internal/users/repo"
	"greenlight/pkg/jsonlog"
	"greenlight/pkg/mailer"
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
	smtp struct {
		host     string
		port     int
		username string
		password string
		sender   string
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

	flag.StringVar(&cfg.smtp.host, "smtp-host", "sandbox.smtp.mailtrap.io", "SMTP host")
	flag.IntVar(&cfg.smtp.port, "smtp-port", 25, "SMTP port")
	flag.StringVar(&cfg.smtp.username, "smtp-username", "470a33d889c91a", "SMTP username")
	flag.StringVar(&cfg.smtp.password, "smtp-password", "7f9cd8765d2352", "SMTP password")
	flag.StringVar(&cfg.smtp.sender, "smtp-sender", "Greenlight <no-reply@greenlight.net>", "SMTP sender")

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

	userHandler := &userHandlers.UserHandler{
		Logger:          logger,
		UserRepo:        userRepos.NewUserSqlxRepo(db),
		TokenRepo:       userRepos.NewTokenSqlxRepo(db),
		PermissionsRepo: permissionsRepo.NewSqlxRepo(db),
		Mailer:          mailer.New(cfg.smtp.host, cfg.smtp.port, cfg.smtp.username, cfg.smtp.password, cfg.smtp.sender),
	}

	tokenHandler := &userHandlers.TokenHandler{
		UserRepo:  userRepos.NewUserSqlxRepo(db),
		TokenRepo: userRepos.NewTokenSqlxRepo(db),
	}

	info := Info{
		healthcheckHandler: healtcheckHandler,
		moviesHandler:      moviesHandler,
		userHandler:        userHandler,
		tokenHandler:       tokenHandler,
		userRepo:           userRepos.NewUserSqlxRepo(db),
		permissionsRepo:    permissionsRepo.NewSqlxRepo(db),
		logger:             logger,
		cfg:                cfg,
	}

	err = Serve(info)
	if err != nil {
		logger.PrintFatal(err, nil)
	}
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
