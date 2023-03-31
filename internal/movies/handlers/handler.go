package handlers

import (
	"errors"
	"fmt"
	"net/http"

	"greenlight/internal/movies/models"
	"greenlight/internal/movies/repositoryerrors"
	"greenlight/pkg/httphelpers"
	"greenlight/pkg/validator"

	"github.com/gin-gonic/gin"
)

type Logger interface {
	Println(v ...interface{})
}

type Repo interface {
	Insert(movie *models.Movie) error
	Get(id int64) (*models.Movie, error)
	Update(movie *models.Movie) error
	Delete(id int64) error
}

type Handler struct {
	Logger Logger
	Repo   Repo
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

	err = h.Repo.Insert(movie)
	if err != nil {
		h.Logger.Println(err.Error())
		httphelpers.StatusInternalServerErrorResponse(c, err)
		return
	}

	headers := make(http.Header)
	headers.Set("Location", fmt.Sprintf("/v1/movies/%d", movie.ID))

	err = httphelpers.WriteJson(c, http.StatusCreated, httphelpers.Envelope{"movie": movie}, headers)
	if err != nil {
		httphelpers.StatusInternalServerErrorResponse(c, err)
	}
}

func (h *Handler) ShowMovie(c *gin.Context) {
	id, err := httphelpers.ReadIDParam(c)
	if err != nil {
		http.NotFound(c.Writer, c.Request)
		return
	}

	movie, err := h.Repo.Get(id)
	if err != nil {
		fmt.Printf("aaa error: %v\n", err)
		switch {
		case errors.Is(err, repositoryerrors.ErrRecordNotFound):
			fmt.Printf("aa1 error: %v\n", err)
			httphelpers.StatusNotFoundResponse(c)
		default:
			fmt.Printf("aa2 error: %v\n", err)
			httphelpers.StatusInternalServerErrorResponse(c, err)
		}
		return
	}

	err = httphelpers.WriteJson(c, http.StatusOK, httphelpers.Envelope{"movie": movie}, nil)
	if err != nil {
		h.Logger.Println(err)
		httphelpers.StatusInternalServerErrorResponse(c, err)
	}
}
