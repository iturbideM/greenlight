package repo

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"greenlight/internal/movies/models"
	"greenlight/internal/repositoryerrors"
	"greenlight/pkg/httphelpers"

	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
)

type sqlxRepo struct {
	db *sqlx.DB
}

func NewSqlxRepo(db *sqlx.DB) *sqlxRepo {
	return &sqlxRepo{
		db: db,
	}
}

func (r *sqlxRepo) Insert(ctx context.Context, movie *models.Movie) error {
	query := `
	INSERT INTO movies (title, year, runtime, genres) 
	VALUES ($1, $2, $3, $4)
	RETURNING id, created_at, version`

	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	row := r.db.QueryRowxContext(ctx, query, movie.Title, movie.Year, movie.Runtime, movie.Genres)
	if err := row.Err(); err != nil {
		return err
	}

	err := row.Scan(&movie.ID, &movie.CreatedAt, &movie.Version)
	if err != nil {
		return err
	}

	return nil
}

func (r *sqlxRepo) Get(id int64) (*models.Movie, error) {
	if id < 1 {
		return nil, repositoryerrors.ErrRecordNotFound
	}

	query := `
	SELECT id, created_at, title, year, runtime, genres, version
	FROM movies
	WHERE id = $1`

	var movie models.Movie

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	err := r.db.GetContext(ctx, &movie, query, id)
	if err != nil {
		return nil, err
	}

	if err != nil {
		switch {
		case errors.Is(err, sql.ErrNoRows):
			return nil, repositoryerrors.ErrRecordNotFound
		default:
			return nil, err
		}
	}

	return &movie, nil
}

func scanMovie(row interface{ Scan(...any) error }, movie *models.Movie) error {
	err := row.Scan(
		&movie.ID,
		&movie.CreatedAt,
		&movie.Title,
		&movie.Year,
		&movie.Runtime,
		&movie.Genres,
		&movie.Version,
	)

	return err
}

func (r *sqlxRepo) Update(movie models.Movie) (models.Movie, error) {
	query := `UPDATE movies
			SET title = $1, year = $2, runtime = $3, genres = $4, version = version + 1
			WHERE id = $5 AND version = $6
			RETURNING version`

	args := []any{
		movie.Title,
		movie.Year,
		movie.Runtime,
		pq.Array(movie.Genres),
		movie.ID,
		movie.Version,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	err := r.db.QueryRowxContext(ctx, query, args...).Scan(&movie.Version)
	if err != nil {
		switch {
		case errors.Is(err, sql.ErrNoRows):
			return models.Movie{}, repositoryerrors.ErrEditConflict
		default:
			return models.Movie{}, err
		}
	}
	return movie, nil
}

func (r *sqlxRepo) Delete(id int64) error {
	if id < 1 {
		return repositoryerrors.ErrRecordNotFound
	}

	query := `DELETE FROM movies WHERE id = $1`

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return repositoryerrors.ErrRecordNotFound
	}

	return nil
}

func (r *sqlxRepo) GetAll(title string, genres []string, filters httphelpers.Filters) ([]*models.Movie, httphelpers.Metadata, error) {
	query := fmt.Sprintf(`
			SELECT count(*) OVER(), id, created_at, title, year, runtime, genres, version
			FROM movies
			WHERE (to_tsvector('simple', title) @@ plainto_tsquery('simple', $1) OR $1 = '')
			AND (genres @> $2 OR $2 = '{}')
			ORDER BY %s %s, id ASC
			LIMIT $3 OFFSET $4`, filters.SortColumn(), filters.SortDirection())

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	args := []any{title, pq.Array(genres), filters.Limit(), filters.Offset()}

	rows, err := r.db.QueryxContext(ctx, query, args...)
	if err != nil {
		return nil, httphelpers.Metadata{}, err
	}

	defer rows.Close()

	totalRecords := 0
	movies := []*models.Movie{}

	for rows.Next() {
		var movie models.Movie

		err := scanMovie(rows, &movie)
		if err != nil {
			return nil, httphelpers.Metadata{}, err
		}

		movies = append(movies, &movie)
	}

	if err := rows.Err(); err != nil {
		return nil, httphelpers.Metadata{}, err
	}

	metadata := httphelpers.CalculateMetadata(totalRecords, filters.Page, filters.PageSize)

	return movies, metadata, nil
}
