package repository

import "errors"

var (
	ErrUserNotFound         = errors.New("user not found")
	ErrUserAlreadyExists    = errors.New("user already exists")
	ErrRefreshTokenNotFound = errors.New("refresh token not found")
	ErrInvalidCredentials   = errors.New("invalid credentials")
)
