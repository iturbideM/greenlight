package handlers

import (
	"fmt"
	"net/http"

	"greenlight/pkg/httphelpers"

	"github.com/gin-gonic/gin"
)

type Handler struct{}

func (h *Handler) CreateMovie(c *gin.Context) {
	fmt.Fprintln(c.Writer, "Create a new movie")
}

func (h *Handler) ShowMovie(c *gin.Context) {
	id, err := httphelpers.ReadIDParam(c)
	if err != nil {
		http.NotFound(c.Writer, c.Request)
		return
	}

	fmt.Fprintf(c.Writer, "Show the details of movie %d\n", id)
}
