package model

import "time"

// storing amounts as int64 kopecks to avoid float precision issues
type Balance struct {
	UserID    int64
	Current   int64
	Withdrawn int64
}

type Withdrawal struct {
	ID          int64
	UserID      int64
	OrderNumber string
	Sum         int64
	ProcessedAt time.Time
}
