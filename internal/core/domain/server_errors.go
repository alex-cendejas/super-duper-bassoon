package domain

import "errors"

var (
	ErrWorkflowNotFound     = errors.New("workflow not found")
	ErrWorkflowInactive     = errors.New("workflow is inactive")
	ErrInvalidWorkflow      = errors.New("invalid workflow")
	ErrClientNotFound       = errors.New("client not found")
	ErrRunNotFound          = errors.New("run not found")
	ErrInvalidFilter        = errors.New("invalid filter expression")
	ErrUnknownField         = errors.New("unknown field")
	ErrTypeMismatch         = errors.New("type mismatch")
	ErrInvalidOperator      = errors.New("invalid operator")
	ErrBanNotFound          = errors.New("ban not found")
	ErrPermanentBan         = errors.New("cannot unban a permanent ban")
	ErrInvalidRequest       = errors.New("invalid request")
	ErrInvalidPolicy        = errors.New("invalid policy")
	ErrCircuitStateNotFound = errors.New("circuit state not found")
	ErrMissingRequiredField = errors.New("missing required field")
)
