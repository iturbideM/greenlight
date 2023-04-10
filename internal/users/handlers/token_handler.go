package handlers

import (
	"errors"
	"net/http"
	"time"

	"greenlight/internal/repositoryerrors"
	"greenlight/internal/users/models"
	"greenlight/pkg/httphelpers"
	"greenlight/pkg/validator"

	"github.com/gin-gonic/gin"
)

type TokenHandler struct {
	UserRepo  UserRepo
	TokenRepo TokenRepo
}

func (h TokenHandler) CreateAuthenticationToken(c *gin.Context) {
	var input struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}

	err := httphelpers.JSONDecode(c, &input)
	if err != nil {
		httphelpers.StatusBadRequestResponse(c, err.Error())
		return
	}

	v := validator.New()

	models.ValidateEmail(v, input.Email)
	models.ValidatePasswordPlaintext(v, input.Password)

	if !v.Valid() {
		httphelpers.StatusUnprocesableEntities(c, v.Errors)
		return
	}

	user, err := h.UserRepo.GetByEmail(c, input.Email)
	if err != nil {
		switch {
		case errors.Is(err, repositoryerrors.ErrRecordNotFound):
			httphelpers.StatusUnauthorizedJSONPayloadResponse(c, "invalid credentials")
		default:
			httphelpers.StatusInternalServerErrorResponse(c, err)
		}
		return
	}

	match, err := user.Password.Matches(input.Password)
	if err != nil {
		httphelpers.StatusInternalServerErrorResponse(c, err)
		return
	}

	if !match {
		httphelpers.StatusUnauthorizedJSONPayloadResponse(c, "invalid credentials")
		return
	}

	token, err := h.TokenRepo.New(user.ID, 24*time.Hour, models.ScopeAuthentication)
	if err != nil {
		httphelpers.StatusInternalServerErrorResponse(c, err)
		return
	}

	httphelpers.CustomStatusJSONPayloadResponse(c, http.StatusOK, gin.H{"token": token}, nil)
	if err != nil {
		httphelpers.StatusInternalServerErrorResponse(c, err)
		return
	}
}
