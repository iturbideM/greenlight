package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	healthcheckHandlers "iturbideM/greenlight/internal/healtcheck/handlers"
	healthcheckRouter "iturbideM/greenlight/internal/healtcheck/router"
	moviesHandlers "iturbideM/greenlight/internal/movies/handlers"
	moviesRouter "iturbideM/greenlight/internal/movies/router"

	"github.com/gin-gonic/gin"
)

const version = "1.0.0"

type config struct {
	port int
	env  string
}

type application struct {
	config config
	logger *log.Logger
}

func main() {
	var cfg config

	flag.IntVar(&cfg.port, "port", 4000, "API server port")
	flag.StringVar(&cfg.env, "env", "development", "Environment (development|staging|production)")
	flag.Parse()

	logger := log.New(os.Stdout, "", log.Ldate|log.Ltime)

	// ? Necesito un application struct para pasarle el logger y la config?
	// app := &application{
	// 	config: cfg,
	// 	logger: logger,
	// }

	healtcheckHandler := &healthcheckHandlers.Handler{
		Version: version,
		Env:     cfg.env,
	}

	moviesHandler := &moviesHandlers.Handler{}

	engine := gin.Default()
	engine.HandleMethodNotAllowed = true
	healthcheckRouter.InitRouter(engine, healtcheckHandler)
	moviesRouter.InitRouter(engine, moviesHandler)

	srv := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.port),
		Handler:      engine,
		IdleTimeout:  time.Minute,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	logger.Printf("starting %s server on %s", cfg.env, srv.Addr)
	err := srv.ListenAndServe()
	logger.Fatal(err)
}
