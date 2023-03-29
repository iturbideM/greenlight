package handlers

import (
	"fmt"
	"net/http"

	"iturbideM/greenlight/pkg/helpers"

	"github.com/gin-gonic/gin"
)

type Handler struct{}

func (h *Handler) CreateMovies(c *gin.Context) {
	fmt.Fprintln(c.Writer, "Create a new movie")
}

func (h *Handler) ShowMovie(c *gin.Context) {
	id, err := helpers.ReadIDParam(c)
	if err != nil {
		http.NotFound(c.Writer, c.Request)
		return
	}

	fmt.Fprintf(c.Writer, "Show the details of the movie: %d\n", id)
}
