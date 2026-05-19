package pubsubpub

import "fmt"

var (
	ErrNotImplemented       = fmt.Errorf("pubsubpub: not implemented")
	ErrClientNotInitialized = fmt.Errorf("pubsubpub: client not initialized")
)

func WrapError(err error, operation string) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("pubsubpub: %s failed: %w", operation, err)
}
