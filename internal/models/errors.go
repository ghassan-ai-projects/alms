package models

import "errors"

// Sentinel errors for ALMS operations.
var (
	// ErrNotFound is returned when a requested resource does not exist.
	ErrNotFound = errors.New("resource not found")

	// ErrConflict is returned when a duplicate resource already exists.
	ErrConflict = errors.New("resource already exists")

	// ErrValidation is returned when input fails validation rules.
	ErrValidation = errors.New("validation error")

	// ErrGapDetected is returned when a sync acknowledgment has gaps.
	ErrGapDetected = errors.New("sync gap detected: missing learnings")
)
