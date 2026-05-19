package memorystore

import "fmt"

var (
	ErrNotImplemented       = fmt.Errorf("memorystore: not implemented")
	ErrClientNotInitialized = fmt.Errorf("memorystore: client not initialized")
	ErrNotFound             = fmt.Errorf("memorystore: key not found")
)

func WrapError(err error, operation string) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("memorystore: %s failed: %w", operation, err)
}
