package services

import (
	"context"
	"errors"
	"time"

	"greenlight/internal/repositoryerrors"
	"greenlight/internal/users/models"
	"greenlight/pkg/mailer"
	"greenlight/pkg/taskutils"
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

type UserService struct {
	UserRepo        UserRepo
	TokenRepo       TokenRepo
	PermissionsRepo PermissionsRepo
	Logger          Logger
	Mailer          mailer.Mailer
}

func NewUserService(userRepo UserRepo, tokenRepo TokenRepo, permissionsRepo PermissionsRepo, logger Logger, mailer mailer.Mailer) *UserService {
	return &UserService{
		UserRepo:        userRepo,
		TokenRepo:       tokenRepo,
		PermissionsRepo: permissionsRepo,
		Logger:          logger,
		Mailer:          mailer,
	}
}

func (s *UserService) RegisterUser(context context.Context, user models.User) (*models.User, error) {
	err := s.UserRepo.Insert(context, &user)
	if err != nil {
		switch {
		case errors.Is(err, repositoryerrors.ErrDuplicateEmail):
			return nil, repositoryerrors.ErrDuplicateEmail
		default:
			return nil, err
		}
	}

	err = s.PermissionsRepo.AddForUser(user.ID, "movies:read")
	if err != nil {
		return nil, err
	}

	token, err := s.TokenRepo.New(user.ID, 24*3*time.Hour, models.ScopeActivation)
	if err != nil {
		return nil, err
	}

	go func() {
		taskutils.BackgroundTask(s.Logger, func() {
			data := map[string]any{
				"activationToken": token.Plaintext,
				"userID":          user.ID,
			}
			err = s.Mailer.Send(user.Email, "user_welcome.tmpl", data)
			if err != nil {
				s.Logger.PrintError(err, nil)
			}
		})
	}()

	return &user, nil
}

func (s *UserService) ActivateUser(context context.Context, tokenPlainText string) (*models.User, error) {
	user, err := s.UserRepo.GetForToken(models.ScopeActivation, tokenPlainText)
	if err != nil {
		return nil, err
	}

	user.Activated = true

	err = s.UserRepo.Update(context, user)
	if err != nil {
		return nil, err
	}

	err = s.TokenRepo.DeleteAllForUser(models.ScopeActivation, user.ID)
	if err != nil {
		return nil, err
	}

	return user, nil
}
