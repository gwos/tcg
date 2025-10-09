package k8s

import "fmt"

var (
	ErrAPI  = fmt.Errorf("API error")
	ErrKAPI = fmt.Errorf("k8s: %w", ErrAPI)
	ErrMAPI = fmt.Errorf("k8s metrics: %w", ErrAPI)
)
