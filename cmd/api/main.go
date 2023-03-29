package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"

	healthcheckHandler "greenlight/internal/healthcheck/handlers"
	healthcheckRouter "greenlight/internal/healthcheck/router"
	moviesHandler "greenlight/internal/movies/handlers"
	moviesRouter "greenlight/internal/movies/router"
)

const version = "1.0.0"

func main() {
	var port int
	var env string

	flag.IntVar(&port, "port", 4000, "API server port")
	flag.StringVar(&env, "env", "development", "Environment (development|staging|production)")
	flag.Parse()

	logger := log.New(os.Stdout, "", log.Ldate|log.Ltime)

	healtcheckHandler := &healthcheckHandler.Handler{
		Version: version,
		Env:     env,
	}

	moviesHandler := &moviesHandler.Handler{}

	engine := gin.Default()
	engine.HandleMethodNotAllowed = true
	healthcheckRouter.InitRouter(engine, healtcheckHandler)
	moviesRouter.InitRouter(engine, moviesHandler)

	srv := &http.Server{
		Addr:         fmt.Sprintf(":%d", port),
		Handler:      engine,
		IdleTimeout:  time.Minute,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	logger.Printf("starting %s server on %s", env, srv.Addr)
	err := srv.ListenAndServe()
	logger.Fatal(err)
}
