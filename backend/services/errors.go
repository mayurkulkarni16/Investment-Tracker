package services

import (
	"errors"
	"fmt"

	"go.mongodb.org/mongo-driver/mongo"
)

// Sentinel errors used by handlers to choose an HTTP status.
var (
	// ErrNotFound: the requested record (or sub-record) doesn't exist for this user.
	ErrNotFound = errors.New("not found")
	// ErrInvalidInput: the request is well-formed JSON but semantically invalid.
	ErrInvalidInput = errors.New("invalid input")
)

// IsNotFound reports whether err means the record doesn't exist (including
// records owned by another user, which repositories report as not found).
func IsNotFound(err error) bool {
	return errors.Is(err, ErrNotFound) || errors.Is(err, mongo.ErrNoDocuments)
}

// IsInvalidInput reports whether err is a client-side input error.
func IsInvalidInput(err error) bool {
	return errors.Is(err, ErrInvalidInput) || errors.Is(err, ErrGoalAllocation)
}

func notFoundf(format string, args ...any) error {
	return fmt.Errorf("%w: %s", ErrNotFound, fmt.Sprintf(format, args...))
}

func invalidf(format string, args ...any) error {
	return fmt.Errorf("%w: %s", ErrInvalidInput, fmt.Sprintf(format, args...))
}
