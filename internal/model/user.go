package model

import "time"

type User struct {
	ID           int64
	Login        string
	PasswordHash string
	CreatedAt    time.Time
}

type UserBalance struct {
	Current   float64
	Withdrawn float64
}
