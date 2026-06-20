package s3

import (
	"fmt"
)

func WrapError(err error) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("failed to execute s3 request: %w", err)
}
