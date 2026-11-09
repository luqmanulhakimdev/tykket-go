package domain

import "errors"

var (
	ErrNotFound           = errors.New("resource not found")
	ErrConflict           = errors.New("resource already exists or conflicts")
	ErrUnauthorized       = errors.New("unauthorized")
	ErrForbidden          = errors.New("forbidden: insufficient permission")
	ErrInvalidInput       = errors.New("invalid input data")
	ErrActiveSprintExists = errors.New("an active sprint already exists for this project")
	ErrInvalidHierarchy   = errors.New("invalid issue hierarchy parent/child relationship")
)

