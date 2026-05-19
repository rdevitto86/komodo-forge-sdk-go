package secretmanager

import "fmt"

var (
	ErrNotImplemented       = fmt.Errorf("secretmanager: not implemented")
	ErrClientNotInitialized = fmt.Errorf("secretmanager: client not initialized")
	ErrNotFound             = fmt.Errorf("secretmanager: secret not found")
)

func WrapError(err error, operation string) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("secretmanager: %s failed: %w", operation, err)
}
