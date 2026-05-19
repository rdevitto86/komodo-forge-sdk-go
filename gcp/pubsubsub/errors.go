package pubsubsub

import "fmt"

var (
	ErrNotImplemented       = fmt.Errorf("pubsubsub: not implemented")
	ErrClientNotInitialized = fmt.Errorf("pubsubsub: client not initialized")
)

func WrapError(err error, operation string) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("pubsubsub: %s failed: %w", operation, err)
}
