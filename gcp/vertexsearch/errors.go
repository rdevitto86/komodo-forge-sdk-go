package vertexsearch

import "fmt"

var (
	ErrNotImplemented       = fmt.Errorf("vertexsearch: not implemented")
	ErrClientNotInitialized = fmt.Errorf("vertexsearch: client not initialized")
)

func WrapError(err error, operation string) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("vertexsearch: %s failed: %w", operation, err)
}
