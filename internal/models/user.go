package models

import "time"

type User struct {
	ID           int64     `json:"id" db:"id" gorm:"primaryKey;autoIncrement"`
	Name         string    `json:"name" db:"name" gorm:"uniqueIndex"`
	RegisteredAt time.Time `json:"registered_at" db:"registered_at" gorm:"autoCreateTime;column:registered_at"`
}
