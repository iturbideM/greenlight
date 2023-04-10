package models

import (
	"database/sql/driver"
	"strings"
	"time"

	"greenlight/pkg/validator"
)

type CustomArray []string

func (a *CustomArray) Scan(value interface{}) error {
	str := string(value.([]byte))
	str = strings.TrimLeft(str, "{")
	str = strings.TrimRight(str, "}")
	strs := strings.Split(str, ",")
	*a = CustomArray(strs)
	return nil
}

func (a CustomArray) Value() (driver.Value, error) {
	strBuilder := strings.Builder{}
	strBuilder.WriteString("{")
	strBuilder.WriteString(strings.Join(a, ","))
	strBuilder.WriteString("}")
	return strBuilder.String(), nil
}

type Movie struct {
	ID        int64        `json:"id" db:"id"`
	CreatedAt time.Time    `json:"-" db:"created_at"`
	Title     string       `json:"title" db:"title"`
	Year      int32        `json:"year,omitempty" db:"year"`
	Runtime   Runtime      `json:"runtime,omitempty" db:"runtime"`
	Genres    *CustomArray `json:"genres,omitempty" db:"genres"`
	Version   int32        `json:"version" db:"version"`
}

func ValidateMovie(v *validator.Validator, movie *Movie) {
	v.Check(movie.Title != "", "title", "must be provided")
	v.Check(len(movie.Title) <= 500, "title", "must not be more than 500 bytes long")

	v.Check(movie.Year != 0, "year", "must be provided")
	v.Check(movie.Year >= 1888, "year", "must be greater than 1888")
	v.Check(movie.Year <= int32(time.Now().Year()), "year", "must not be in the future")

	v.Check(movie.Runtime != 0, "runtime", "must be provided")
	v.Check(movie.Runtime > 0, "runtime", "must be a positive integer")

	v.Check(movie.Genres != nil, "genres", "must be provided")
	v.Check(len(*movie.Genres) >= 1, "genres", "must contain at least 1 genre")
	v.Check(len(*movie.Genres) <= 5, "genres", "must not contain more than 5 genres")
	v.Check(validator.Unique(*movie.Genres), "genres", "must not contain duplicate values")
}
