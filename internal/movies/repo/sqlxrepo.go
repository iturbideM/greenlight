package repo

import (
	"database/sql"
	"errors"

	"greenlight/internal/movies/models"
	"greenlight/internal/movies/repositoryerrors"

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

func (r *sqlxRepo) Insert(movie *models.Movie) error {
	query := `
	INSERT INTO movies (title, year, runtime, genres) 
	VALUES ($1, $2, $3, $4)
	RETURNING id, created_at, version`

	row := r.db.QueryRowx(query, movie.Title, movie.Year, movie.Runtime, pq.Array(movie.Genres))
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

	row := r.db.QueryRowx(query, id)
	if err := row.Err(); err != nil {
		return nil, err
	}

	err := row.Scan(
		&movie.ID,
		&movie.CreatedAt,
		&movie.Title,
		&movie.Year,
		&movie.Runtime,
		pq.Array(&movie.Genres),
		&movie.Version,
	)
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

func (r *sqlxRepo) Update(movie *models.Movie) error {
	return nil
}

func (r *sqlxRepo) Delete(id int64) error {
	return nil
}
