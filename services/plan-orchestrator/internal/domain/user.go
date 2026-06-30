package domain

import "time"

type User struct {
	ID           int64
	Username     string
	PasswordHash string
	Nickname     string
	Status       int
	CreatedAt    time.Time
	UpdatedAt    time.Time
}
