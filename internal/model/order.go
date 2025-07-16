package model

import "time"

type Order struct {
	Number     string    `json:"number"`
	UserID     int64     `json:"-"`
	Status     string    `json:"status"`
	Accrual    float64   `json:"accrual,omitempty"`
	UploadedAt time.Time `json:"uploaded_at"`
}
