package repo

import (
	"context"
	"time"

	"greenlight/internal/users/models"

	"github.com/jmoiron/sqlx"
)

type TokenRepo struct {
	DB *sqlx.DB
}

func NewTokenSqlxRepo(db *sqlx.DB) *TokenRepo {
	return &TokenRepo{
		DB: db,
	}
}

func (r TokenRepo) New(userID int64, ttl time.Duration, scope string) (*models.Token, error) {
	token, err := models.GenerateToken(userID, ttl, scope)
	if err != nil {
		return nil, err
	}

	err = r.Insert(token)

	return token, err
}

func (r *TokenRepo) Insert(token *models.Token) error {
	query := `
	INSERT INTO tokens (hash, user_id, expiry, scope)
	VALUES ($1, $2, $3, $4)
	`

	args := []any{token.Hash, token.UserID, token.Expiry, token.Scope}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	_, err := r.DB.ExecContext(ctx, query, args...)
	return err
}

func (r *TokenRepo) DeleteAllForUser(scope string, userID int64) error {
	query := `
	DELETE FROM tokens
	WHERE scope = $1 AND user_id = $2
	`

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	_, err := r.DB.ExecContext(ctx, query, scope, userID)
	return err
}
