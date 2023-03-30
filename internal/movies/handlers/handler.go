package handlers

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"greenlight/internal/movies/models"
	"greenlight/pkg/httphelpers"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	Logger *log.Logger
}

func (h *Handler) CreateMovie(c *gin.Context) {
	fmt.Fprintln(c.Writer, "Create a new movie")
}

func (h *Handler) ShowMovie(c *gin.Context) {
	id, err := httphelpers.ReadIDParam(c)
	if err != nil {
		http.NotFound(c.Writer, c.Request)
		return
	}

	movie := models.Movie{
		ID:        id,
		CreatedAt: time.Now(),
		Title:     "Casablanca",
		Runtime:   102,
		Genres:    []string{"Drama", "Romance", "War"},
		Version:   1,
	}

	err = httphelpers.WriteJson(c, http.StatusOK, httphelpers.Envelope{"movie": movie}, nil)
	if err != nil {
		h.Logger.Println(err)
		httphelpers.StatusInternalServerErrorResponse(c, err)
	}
}
