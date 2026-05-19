package cloudlogging

import "fmt"

var (
	ErrNotImplemented       = fmt.Errorf("cloudlogging: not implemented")
	ErrClientNotInitialized = fmt.Errorf("cloudlogging: client not initialized")
)

func WrapError(err error, operation string) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("cloudlogging: %s failed: %w", operation, err)
}
