package model

import "time"

const (
	OrderStatusNew        = "NEW"
	OrderStatusProcessing = "PROCESSING"
	OrderStatusProcessed  = "PROCESSED"
	OrderStatusInvalid    = "INVALID"
)

type Order struct {
	ID          int64
	UserID      int64
	OrderNumber string
	Status      string
	Accrual     *int64
	UploadedAt  time.Time
}
