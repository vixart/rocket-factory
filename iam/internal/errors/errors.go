package errs

import "errors"

var (
	// User errors.
	ErrUserNotFound       = errors.New("user not found")
	ErrUserAlreadyExists  = errors.New("a user with this login already exists")
	ErrInvalidCredentials = errors.New("invalid login or password")

	// Session errors.
	ErrSessionNotFound = errors.New("session not found or expired")

	// Validation errors.
	ErrInvalidLogin    = errors.New("login is required")
	ErrWeakPassword    = errors.New("password must be at least 8 characters long")
	ErrEmptyCredential = errors.New("login and password are required")
	ErrEmptySessionID  = errors.New("session_uuid is required")
	ErrInvalidUUID     = errors.New("invalid UUID format")
)
