package model

import "time"

type Withdrawal struct {
	Order       string
	Sum         float64
	UserID      int64
	ProcessedAt time.Time
}
