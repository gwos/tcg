package clients

import (
	"errors"
)

// define error types used in dispatcher
var (
	ErrGateway      = errors.New("gateway error")
	ErrSynchronizer = errors.New("synchronizer error")
	ErrUnauthorized = errors.New("unauthorized")
	ErrUndecided    = errors.New("undecided error")
)
