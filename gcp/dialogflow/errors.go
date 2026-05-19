package dialogflow

import "fmt"

var (
	ErrNotImplemented       = fmt.Errorf("dialogflow: not implemented")
	ErrClientNotInitialized = fmt.Errorf("dialogflow: client not initialized")
)

func WrapError(err error, operation string) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("dialogflow: %s failed: %w", operation, err)
}
