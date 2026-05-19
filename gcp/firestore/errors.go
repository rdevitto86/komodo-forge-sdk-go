package firestore

import "fmt"

var (
	ErrNotImplemented       = fmt.Errorf("firestore: not implemented")
	ErrClientNotInitialized = fmt.Errorf("firestore: client not initialized")
	ErrNotFound             = fmt.Errorf("firestore: document not found")
)

func WrapError(err error, operation string) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("firestore: %s failed: %w", operation, err)
}
