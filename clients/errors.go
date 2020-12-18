package clients

import (
	"fmt"

	tcgerr "github.com/gwos/tcg/errors"
)

// define custom error types
var (
	ErrGateway      = fmt.Errorf("%w: %v", tcgerr.ErrTransient, "gateway error")
	ErrSynchronizer = fmt.Errorf("%w: %v", tcgerr.ErrTransient, "synchronizer error")
	ErrUnauthorized = fmt.Errorf("%w: %v", tcgerr.ErrPermanent, "unauthorized")
	ErrUndecided    = fmt.Errorf("%w: %v", tcgerr.ErrPermanent, "undecided error")
)
