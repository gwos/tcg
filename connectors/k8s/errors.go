package k8s

import (
	"fmt"

	k8serrors "k8s.io/apimachinery/pkg/api/errors"
)

var (
	ErrAPI  = fmt.Errorf("API error")
	ErrKAPI = fmt.Errorf("k8s: %w", ErrAPI)
	ErrMAPI = fmt.Errorf("k8s metrics: %w", ErrAPI)
)

func IsAuthError(err error) bool {
	return k8serrors.IsUnauthorized(err) || k8serrors.IsForbidden(err)
}

func IsMetricsUnavailable(err error) bool {
	return k8serrors.IsNotFound(err) || k8serrors.IsServiceUnavailable(err)
}
