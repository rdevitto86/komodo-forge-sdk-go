package dynamodb

import (
	"errors"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

var ErrNotFound = errors.New("item not found")

func WrapError(err error) error {
	if err == nil {
		return nil
	}

	var (
		conditionalCheckErr      *types.ConditionalCheckFailedException
		resourceNotFoundErr      *types.ResourceNotFoundException
		provisionedThroughputErr *types.ProvisionedThroughputExceededException
		requestLimitErr          *types.RequestLimitExceeded
		transactionConflictErr   *types.TransactionConflictException
		transactionCanceledErr   *types.TransactionCanceledException
		duplicateItemErr         *types.DuplicateItemException
		resourceInUseErr         *types.ResourceInUseException
		tableNotFoundErr         *types.TableNotFoundException
		internalServerErr        *types.InternalServerError
		itemCollectionErr        *types.ItemCollectionSizeLimitExceededException
		throttlingErr            *types.ThrottlingException
	)

	switch {
	case errors.As(err, &conditionalCheckErr):
		return fmt.Errorf("failed conditional check: %w", err)

	case errors.As(err, &resourceNotFoundErr):
		return fmt.Errorf("failed to find resource: %w", err)

	case errors.As(err, &provisionedThroughputErr):
		return fmt.Errorf("exceeded provisioned throughput: %w", err)

	case errors.As(err, &requestLimitErr):
		return fmt.Errorf("exceeded request limit: %w", err)

	case errors.As(err, &transactionConflictErr):
		return fmt.Errorf("hit transaction conflict: %w", err)

	case errors.As(err, &transactionCanceledErr):
		return fmt.Errorf("canceled transaction: %w", err)

	case errors.As(err, &duplicateItemErr):
		return fmt.Errorf("rejected duplicate item: %w", err)

	case errors.As(err, &resourceInUseErr):
		return fmt.Errorf("found resource in use: %w", err)

	case errors.As(err, &tableNotFoundErr):
		return fmt.Errorf("failed to find table: %w", err)

	case errors.As(err, &internalServerErr):
		return fmt.Errorf("hit internal server error: %w", err)

	case errors.As(err, &itemCollectionErr):
		return fmt.Errorf("exceeded item collection size: %w", err)

	case errors.As(err, &throttlingErr):
		return fmt.Errorf("throttled request: %w", err)

	default:
		return fmt.Errorf("failed dynamodb request: %w", err)
	}
}
