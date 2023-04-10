package handlers

import (
	"context"
	"errors"
	"net/http"
	"time"

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

type UserRepo interface {
	Insert(context.Context, *models.User) error
	GetByEmail(ctx context.Context, email string) (*models.User, error)
	Update(ctx context.Context, user *models.User) error
	GetForToken(tokenScope, tokenPlaintext string) (*models.User, error)
}

type TokenRepo interface {
	New(userID int64, ttl time.Duration, scope string) (*models.Token, error)
	Insert(token *models.Token) error
	DeleteAllForUser(scope string, userID int64) error
}

type PermissionsRepo interface {
	AddForUser(userID int64, codes ...string) error
}

type UserService interface {
	RegisterUser(context context.Context, user models.User) (*models.User, error)
	ActivateUser(context context.Context, tokenPlaintext string) (*models.User, error)
}

type UserHandler struct {
	UserRepo        UserRepo
	TokenRepo       TokenRepo
	PermissionsRepo PermissionsRepo
	Logger          Logger
	Mailer          mailer.Mailer
	UserService     UserService
}

type UserRegisterInput struct {
	Name     string `json:"name"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (h *UserHandler) Register(c *gin.Context) {
	var input UserRegisterInput

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

	user, err = h.UserService.RegisterUser(c, *user)
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

	err = httphelpers.CustomStatusJSONPayloadResponse(c, http.StatusCreated, gin.H{"user": user}, nil)
	if err != nil {
		httphelpers.StatusInternalServerErrorResponse(c, err)
	}
}

type ActivateUserInput struct {
	TokenPlaintext string `json:"token"`
}

func (h *UserHandler) ActivateUser(c *gin.Context) {
	var input ActivateUserInput
	err := httphelpers.JSONDecode(c, &input)
	if err != nil {
		httphelpers.StatusBadRequestResponse(c, err.Error())
		return
	}

	v := validator.New()

	if models.ValidateTokenPlaintext(v, input.TokenPlaintext); !v.Valid() {
		httphelpers.StatusUnprocesableEntities(c, v.Errors)
		return
	}

	user, err := h.UserService.ActivateUser(c, input.TokenPlaintext)
	if err != nil {
		switch {
		case errors.Is(err, repositoryerrors.ErrRecordNotFound):
			v.AddError("token", "invalid or expired activation token")
			httphelpers.StatusUnprocesableEntities(c, v.Errors)
		case errors.Is(err, repositoryerrors.ErrEditConflict):
			httphelpers.StatusConflictResponse(c)
		default:
			httphelpers.StatusInternalServerErrorResponse(c, err)
		}
	}

	err = httphelpers.CustomStatusJSONPayloadResponse(c, http.StatusOK, gin.H{"user": user}, nil)
	if err != nil {
		httphelpers.StatusInternalServerErrorResponse(c, err)
	}
}
