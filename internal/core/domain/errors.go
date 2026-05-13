package domain

import "errors"

var (
	ErrClientNotFound  = errors.New("client not found")
	ErrInvalidActivity = errors.New("invalid activity type")
	ErrCrippledClient  = errors.New("client is crippled")
	ErrStateConflict   = errors.New("state conflict")
)
