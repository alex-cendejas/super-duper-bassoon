package domain

import "errors"

var (
	ErrInnerClientNotFound = errors.New("inner client not found")
	ErrInvalidActivity     = errors.New("invalid activity type")
	ErrCrippledClient      = errors.New("client is crippled")
	ErrStateConflict       = errors.New("state conflict")
)
