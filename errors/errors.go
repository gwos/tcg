package errors

import (
	"errors"
)

// define error types for retry logic
var (
	ErrPermanent = errors.New("permanent error")
	ErrTransient = errors.New("transient error")
)
