package cloudfunctions

import "fmt"

var (
	ErrNotImplemented       = fmt.Errorf("cloudfunctions: not implemented")
	ErrClientNotInitialized = fmt.Errorf("cloudfunctions: client not initialized")
)

func WrapError(err error, operation string) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("cloudfunctions: %s failed: %w", operation, err)
}
