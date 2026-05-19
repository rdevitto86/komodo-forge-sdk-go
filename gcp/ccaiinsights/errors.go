package ccaiinsights

import "fmt"

var (
	ErrNotImplemented       = fmt.Errorf("ccaiinsights: not implemented")
	ErrClientNotInitialized = fmt.Errorf("ccaiinsights: client not initialized")
)

func WrapError(err error, operation string) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("ccaiinsights: %s failed: %w", operation, err)
}
