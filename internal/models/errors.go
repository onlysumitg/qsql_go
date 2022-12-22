package models

import (
	"errors"
)

var (
	ErrNoRecord = errors.New("models: no matching record found")
	// Add a new ErrInvalidCredentials error. We'll use this later if a user
	// tries to login with an incorrect email address or password.
	ErrInvalidCredentials = errors.New("models: invalid credentials")
	// Add a new ErrDuplicateEmail error. We'll use this later if a user
	// tries to signup with an email address that's already in use.
	ErrDuplicateEmail = errors.New("models: duplicate email")

	ErrUserNotFound       = errors.New("models: User not found")
	ErrServerNotFound     = errors.New("models: Server not found")
	ErrSavedQueryNotFound = errors.New("models: Saved query not found")
)
