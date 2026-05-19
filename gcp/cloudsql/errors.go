package cloudsql

import "fmt"

var (
	ErrNotImplemented       = fmt.Errorf("cloudsql: not implemented")
	ErrClientNotInitialized = fmt.Errorf("cloudsql: client not initialized")
)

func WrapError(err error, operation string) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("cloudsql: %s failed: %w", operation, err)
}
