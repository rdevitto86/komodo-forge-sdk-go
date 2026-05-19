package gcs

import "fmt"

var (
	ErrNotImplemented       = fmt.Errorf("gcs: not implemented")
	ErrClientNotInitialized = fmt.Errorf("gcs: client not initialized")
)

func WrapError(err error, operation string) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("gcs: %s failed: %w", operation, err)
}
