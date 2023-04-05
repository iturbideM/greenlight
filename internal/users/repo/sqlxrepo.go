package repo

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"greenlight/internal/repositoryerrors"
	"greenlight/internal/users/models"

	"github.com/jmoiron/sqlx"
)

type Repo struct {
	db *sqlx.DB
}

func NewSqlxRepo(db *sqlx.DB) *Repo {
	return &Repo{
		db: db,
	}
}

func (r *Repo) Insert(ctx context.Context, user *models.User) error {
	query := `
	INSERT INTO users (name, email, password_hash, activated)
	VALUES ($1, $2, $3, $4)
	RETURNING id, created_at, version`

	args := []any{user.Name, user.Email, user.Password.Hash(), user.Activated}

	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	err := r.db.QueryRowxContext(ctx, query, args...).Scan(&user.ID, &user.CreatedAt, &user.Version)
	if err != nil {
		switch {
		case err.Error() == `pq: duplicate key value violates unique constraint "users_email_key"`:
			return repositoryerrors.ErrDuplicateEmail
		default:
			return err
		}
	}

	return nil
}

func (r *Repo) GetByEmail(ctx context.Context, email string) (*models.User, error) {
	query := `
		SELECT id, created_at, name, email, password_hash, activated, version
		FROM users
		WHERE email = $1`

	var user models.User

	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	err := r.db.GetContext(ctx, &user, query, email)
	if err != nil {
		switch {
		case errors.Is(err, sql.ErrNoRows):
			return nil, repositoryerrors.ErrRecordNotFound
		default:
			return nil, err
		}
	}

	return &user, nil
}

func (r *Repo) Update(ctx context.Context, user *models.User) error {
	query := `
		UPDATE users
		SET name = $1, email = $2, password_hash = $3, activated = $4, version = version + 1
		WHERE id = $5 AND version = $6
		RETURNING version`

	args := []any{
		user.Name,
		user.Email,
		user.Password.Hash,
		user.Activated,
		user.ID,
		user.Version,
	}

	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	err := r.db.QueryRowxContext(ctx, query, args...).Scan(&user.Version)
	if err != nil {
		switch {
		case err.Error() == `pq: duplicate key value violates unique constraint "users_email_key"`:
			return repositoryerrors.ErrDuplicateEmail
		case errors.Is(err, sql.ErrNoRows):
			return repositoryerrors.ErrEditConflict
		default:
			return err
		}
	}

	return nil
}
