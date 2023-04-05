package handlers

import (
	"context"
	"errors"
	"net/http"

	"greenlight/internal/repositoryerrors"
	"greenlight/internal/users/models"
	"greenlight/pkg/httphelpers"
	"greenlight/pkg/mailer"
	"greenlight/pkg/validator"

	"github.com/gin-gonic/gin"
)

type Logger interface {
	PrintInfo(message string, properties map[string]string)
	PrintError(err error, properties map[string]string)
	PrintFatal(err error, properties map[string]string)
}

type Repo interface {
	Insert(context.Context, *models.User) error
	GetByEmail(ctx context.Context, email string) (*models.User, error)
	Update(ctx context.Context, user *models.User) error
}

type Handler struct {
	Repo   Repo
	Logger Logger
	Mailer mailer.Mailer
}

func (h *Handler) Register(c *gin.Context) {
	var input struct {
		Name     string `json:"name"`
		Email    string `json:"email"`
		Password string `json:"password"`
	}

	err := httphelpers.JSONDecode(c, &input)
	if err != nil {
		httphelpers.StatusBadRequestResponse(c, err.Error())
		return
	}

	user := &models.User{
		Name:      input.Name,
		Email:     input.Email,
		Activated: false,
	}

	err = user.Password.Set(input.Password)
	if err != nil {
		httphelpers.StatusInternalServerErrorResponse(c, err)
		return
	}

	v := validator.New()

	if models.ValidateUser(v, user); !v.Valid() {
		httphelpers.StatusUnprocesableEntities(c, v.Errors)
		return
	}

	err = h.Repo.Insert(c, user)
	if err != nil {
		switch {
		case errors.Is(err, repositoryerrors.ErrDuplicateEmail):
			v.AddError("email", "a user with that email address already exists")
			httphelpers.StatusUnprocesableEntities(c, v.Errors)
		default:
			httphelpers.StatusInternalServerErrorResponse(c, err)
		}

		return
	}

	err = h.Mailer.Send(user.Email, "user_welcome.tmpl", user)
	if err != nil {
		httphelpers.StatusInternalServerErrorResponse(c, err)
		return
	}

	httphelpers.WriteJson(c, http.StatusCreated, httphelpers.Envelope{"user": user}, nil)
	if err != nil {
		httphelpers.StatusInternalServerErrorResponse(c, err)
	}
}
