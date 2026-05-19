package cloudmonitoring

import "fmt"

var (
	ErrNotImplemented       = fmt.Errorf("cloudmonitoring: not implemented")
	ErrClientNotInitialized = fmt.Errorf("cloudmonitoring: client not initialized")
)

func WrapError(err error, operation string) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("cloudmonitoring: %s failed: %w", operation, err)
}
