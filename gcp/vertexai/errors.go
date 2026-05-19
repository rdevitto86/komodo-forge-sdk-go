package vertexai

import "fmt"

var (
	ErrNotImplemented       = fmt.Errorf("vertexai: not implemented")
	ErrClientNotInitialized = fmt.Errorf("vertexai: client not initialized")
)

func WrapError(err error, operation string) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("vertexai: %s failed: %w", operation, err)
}
