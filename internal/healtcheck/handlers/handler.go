package handlers

import (
	"fmt"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	Version string
	Env     string
}

func (h *Handler) HealthcheckHandler(c *gin.Context) {
	fmt.Fprintln(c.Writer, "status: available")
	fmt.Fprintf(c.Writer, "environment: %s\n", h.Env)
	fmt.Fprintf(c.Writer, "version: %s\n", h.Version)
}
