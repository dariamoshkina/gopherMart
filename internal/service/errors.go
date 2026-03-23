package service

import "errors"

var (
	ErrInvalidCredentials    = errors.New("invalid credentials")
	ErrLoginTaken            = errors.New("login already taken")
	ErrUserNotFound          = errors.New("user not found")
	ErrInvalidOrderNumber    = errors.New("invalid order number")
	ErrDuplicateOrderNumber  = errors.New("order number already submitted")
	ErrOrderOwnedBySameUser  = errors.New("order already submitted by this user")
	ErrOrderOwnedByOtherUser = errors.New("order already submitted by another user")
	ErrInsufficientBalance   = errors.New("insufficient balance")
)
