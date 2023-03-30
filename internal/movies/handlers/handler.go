package handlers

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"greenlight/internal/movies/models"
	"greenlight/pkg/httphelpers"
	"greenlight/pkg/validator"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	Logger *log.Logger
}

func (h *Handler) CreateMovie(c *gin.Context) {
	var input struct {
		Title   string         `json:"title"`
		Year    int32          `json:"year"`
		Runtime models.Runtime `json:"runtime"`
		Genres  []string       `json:"genres"`
	}

	err := httphelpers.JSONDecode(c, &input)
	if err != nil {
		h.Logger.Println(err.Error())
		httphelpers.StatusBadRequestResponse(c, err.Error())
		return
	}

	movie := &models.Movie{
		Title:   input.Title,
		Year:    input.Year,
		Runtime: input.Runtime,
		Genres:  input.Genres,
	}

	v := validator.New()

	if models.ValidateMovie(v, movie); !v.Valid() {
		httphelpers.StatusUnprocesableEntities(c, v.Errors)
		return
	}

	fmt.Fprintf(c.Writer, "%+v\n", input)
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
