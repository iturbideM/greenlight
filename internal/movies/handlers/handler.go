package handlers

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"greenlight/internal/movies/models"
	"greenlight/internal/repositoryerrors"
	"greenlight/pkg/httphelpers"
	"greenlight/pkg/validator"

	"github.com/gin-gonic/gin"
)

type Logger interface {
	PrintInfo(message string, properties map[string]string)
	PrintError(err error, properties map[string]string)
	PrintFatal(err error, properties map[string]string)
}

type Repo interface {
	Insert(context.Context, *models.Movie) error
	Get(id int64) (*models.Movie, error)
	GetAll(title string, genres []string, filters httphelpers.Filters) ([]*models.Movie, httphelpers.Metadata, error)
	Update(movie models.Movie) (models.Movie, error)
	Delete(id int64) error
}

type Handler struct {
	Logger Logger
	Repo   Repo
}

type createMovieInput struct {
	Title   string              `json:"title"`
	Year    int32               `json:"year"`
	Runtime models.Runtime      `json:"runtime"`
	Genres  *models.CustomArray `json:"genres"`
}

func (h *Handler) CreateMovie(c *gin.Context) {
	var input createMovieInput
	err := httphelpers.JSONDecode(c, &input)
	if err != nil {
		h.Logger.PrintError(err, nil)
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

	ctx := context.Background()

	err = h.Repo.Insert(ctx, movie)
	if err != nil {
		httphelpers.StatusInternalServerErrorResponse(c, err)
		return
	}

	headers := make(http.Header)
	headers.Set("Location", fmt.Sprintf("/v1/movies/%d", movie.ID))

	err = httphelpers.CustomStatusJSONPayloadResponse(c, http.StatusCreated, gin.H{"movie": movie}, headers)
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
		switch {
		case errors.Is(err, repositoryerrors.ErrRecordNotFound):
			httphelpers.StatusNotFoundResponse(c)
		default:
			httphelpers.StatusInternalServerErrorResponse(c, err)
		}
		return
	}

	err = httphelpers.CustomStatusJSONPayloadResponse(c, http.StatusOK, gin.H{"movie": movie}, nil)
	if err != nil {
		httphelpers.StatusInternalServerErrorResponse(c, err)
	}
}

type updateMovieInput struct {
	Title   *string             `json:"title"`
	Year    *int32              `json:"year"`
	Runtime *models.Runtime     `json:"runtime"`
	Genres  *models.CustomArray `json:"genres"`
}

func (h *Handler) UpdateMovie(c *gin.Context) {
	var input updateMovieInput
	id, err := httphelpers.ReadIDParam(c)
	if err != nil {
		httphelpers.StatusNotFoundResponse(c)
		return
	}

	movie, err := h.Repo.Get(id)
	if err != nil {
		switch {
		case errors.Is(err, repositoryerrors.ErrRecordNotFound):
			httphelpers.StatusNotFoundResponse(c)
		default:
			httphelpers.StatusInternalServerErrorResponse(c, err)
		}
		return
	}

	err = httphelpers.JSONDecode(c, &input)
	if err != nil {
		httphelpers.StatusBadRequestResponse(c, err.Error())
		return
	}

	if input.Title != nil {
		movie.Title = *input.Title
	}
	if input.Year != nil {
		movie.Year = *input.Year
	}
	if input.Runtime != nil {
		movie.Runtime = *input.Runtime
	}
	if input.Genres != nil {
		movie.Genres = input.Genres
	}

	v := validator.New()

	if models.ValidateMovie(v, movie); !v.Valid() {
		httphelpers.StatusUnprocesableEntities(c, v.Errors)
		return
	}

	*movie, err = h.Repo.Update(*movie)
	if err != nil {
		switch {
		case errors.Is(err, repositoryerrors.ErrEditConflict):
			httphelpers.StatusConflictResponse(c)
		default:
			httphelpers.StatusInternalServerErrorResponse(c, err)
		}
		return
	}

	err = httphelpers.CustomStatusJSONPayloadResponse(c, http.StatusOK, gin.H{"movie": movie}, nil)
	if err != nil {
		httphelpers.StatusInternalServerErrorResponse(c, err)
	}
}

func (h *Handler) DeleteMovie(c *gin.Context) {
	id, err := httphelpers.ReadIDParam(c)
	if err != nil {
		httphelpers.StatusNotFoundResponse(c)
		return
	}

	err = h.Repo.Delete(id)
	if err != nil {
		switch {
		case errors.Is(err, repositoryerrors.ErrRecordNotFound):
			httphelpers.StatusNotFoundResponse(c)
		default:
			httphelpers.StatusInternalServerErrorResponse(c, err)
		}
		return
	}

	err = httphelpers.CustomStatusJSONPayloadResponse(c, http.StatusOK, gin.H{"message": "movie successfully deleted"}, nil)
	if err != nil {
		httphelpers.StatusInternalServerErrorResponse(c, err)
	}
}

type listMoviesInput struct {
	Title  string
	Genres []string
	httphelpers.Filters
}

func (h *Handler) ListMovies(c *gin.Context) {
	v := validator.New()

	qs := c.Request.URL.Query()

	input := listMoviesInput{
		Title:  httphelpers.ReadString(qs, "title", ""),
		Genres: httphelpers.ReadCSV(qs, "genres", []string{}),
		Filters: httphelpers.Filters{
			Page:         httphelpers.ReadInt(qs, "page", 1, v),
			PageSize:     httphelpers.ReadInt(qs, "page_size", 10, v),
			Sort:         httphelpers.ReadString(qs, "sort", "id"),
			SortSafeList: []string{"id", "title", "year", "runtime", "-id", "-title", "-year", "-runtime"},
		},
	}

	if httphelpers.ValidateFilters(v, input.Filters); !v.Valid() {
		httphelpers.StatusUnprocesableEntities(c, v.Errors)
		return
	}

	movies, metadata, err := h.Repo.GetAll(input.Title, input.Genres, input.Filters)
	if err != nil {
		httphelpers.StatusInternalServerErrorResponse(c, err)
		return
	}

	err = httphelpers.CustomStatusJSONPayloadResponse(c, http.StatusOK, gin.H{"movies": movies, "metadata": metadata}, nil)
	if err != nil {
		httphelpers.StatusInternalServerErrorResponse(c, err)
	}
}
