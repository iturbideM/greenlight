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
	"greenlight/pkg/taskutils"
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

type UserHandler struct {
	UserRepo        UserRepo
	TokenRepo       TokenRepo
	PermissionsRepo PermissionsRepo
	Logger          Logger
	Mailer          mailer.Mailer
}

func (h *UserHandler) Register(c *gin.Context) {
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

	// y la capa de servicio/logica??
	// esto refactorealo a otro paquete
	err = h.UserRepo.Insert(c, user)
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

	err = h.PermissionsRepo.AddForUser(user.ID, "movies:read")
	if err != nil {
		httphelpers.StatusInternalServerErrorResponse(c, err)
		return
	}

	token, err := h.TokenRepo.New(user.ID, 24*3*time.Hour, models.ScopeActivation)
	if err != nil {
		httphelpers.StatusInternalServerErrorResponse(c, err)
		return
	}

	go func() {
		taskutils.BackgroundTask(h.Logger, func() {
			data := map[string]any{
				"activationToken": token.Plaintext,
				"userID":          user.ID,
			}
			err = h.Mailer.Send(user.Email, "user_welcome.tmpl", data)
			if err != nil {
				h.Logger.PrintError(err, nil)
			}
		})
	}()
	// hasta aca va todo en capa de servicio, devolviendo el error correspondiente a cada caso
	err = httphelpers.WriteJson(c, http.StatusCreated, gin.H{"user": user}, nil)
	if err != nil {
		httphelpers.StatusInternalServerErrorResponse(c, err)
	}
}

func (h *UserHandler) ActivateUser(c *gin.Context) {
	var input struct {
		TokenPlaintext string `json:"token"`
	}

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

	// capa de servicio
	user, err := h.UserRepo.GetForToken(models.ScopeActivation, input.TokenPlaintext)
	if err != nil {
		switch {
		case errors.Is(err, repositoryerrors.ErrRecordNotFound):
			v.AddError("token", "invalid or expired activation token")
			httphelpers.StatusUnprocesableEntities(c, v.Errors)
		default:
			httphelpers.StatusInternalServerErrorResponse(c, err)
		}
		return
	}

	user.Activated = true

	err = h.UserRepo.Update(c, user)
	if err != nil {
		switch {
		case errors.Is(err, repositoryerrors.ErrEditConflict):
			httphelpers.StatusConflictResponse(c)
		default:
			httphelpers.StatusInternalServerErrorResponse(c, err)
		}
		return
	}

	err = h.TokenRepo.DeleteAllForUser(models.ScopeActivation, user.ID)
	if err != nil {
		httphelpers.StatusInternalServerErrorResponse(c, err)
		return
	}
	// hasta aca en capa de servicio

	err = httphelpers.WriteJson(c, http.StatusOK, gin.H{"user": user}, nil)
	if err != nil {
		httphelpers.StatusInternalServerErrorResponse(c, err)
	}
}
