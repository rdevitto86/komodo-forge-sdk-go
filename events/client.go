package events

import (
	"context"
	"encoding/json"
	"fmt"

	logger "github.com/rdevitto86/komodo-forge-sdk-go/logging/runtime"

	"github.com/rdevitto86/komodo-forge-sdk-go/aws/sns"
	"github.com/rdevitto86/komodo-forge-sdk-go/aws/sqs"
)

// Publisher publishes Events to an SNS FIFO topic.
type Publisher struct {
	sns      sns.API
	topicARN string
}

// NewPublisher creates a Publisher that publishes to the given SNS topic ARN.
func NewPublisher(snsClient sns.API, topicARN string) *Publisher {
	return &Publisher{sns: snsClient, topicARN: topicARN}
}

// Publish JSON-encodes the event and publishes it to the SNS FIFO topic.
// MessageGroupId is set to event.EntityID; MessageDeduplicationId to event.ID,
// which guarantees ordering per entity and idempotent delivery.
func (p *Publisher) Publish(ctx context.Context, event Event) error {
	body, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("events: marshal failed: %w", err)
	}

	_, err = p.sns.Publish(ctx, sns.PublishInput{
		TopicARN: p.topicARN,
		Message:  string(body),
		GroupID:  event.EntityID,
		DedupID:  event.ID,
	})
	if err != nil {
		logger.Error(fmt.Sprintf("events: publish failed for event %s (%s)", event.ID, event.Type), err)
		return err
	}

	logger.Info(fmt.Sprintf("events: published %s id=%s entity=%s", event.Type, event.ID, event.EntityID))
	return nil
}

// SubscriberConfig configures a Subscriber.
type SubscriberConfig struct {
	// QueueURL is the SQS queue URL to poll.
	QueueURL string
	// MaxBatch is the maximum number of messages to receive per poll (1–10, default 10).
	MaxBatch int32
}

// Subscriber consumes Events from an SQS FIFO queue via long-poll.
type Subscriber struct {
	sqs      sqs.API
	queueURL string
	maxBatch int32
}

// NewSubscriber creates a Subscriber that reads from the given SQS queue.
func NewSubscriber(sqsClient sqs.API, cfg SubscriberConfig) *Subscriber {
	batch := cfg.MaxBatch
	if batch <= 0 || batch > 10 {
		batch = 10
	}
	return &Subscriber{
		sqs:      sqsClient,
		queueURL: cfg.QueueURL,
		maxBatch: batch,
	}
}

// Subscribe runs a long-poll loop until ctx is cancelled. For each received message:
//   - Unmarshal the Event envelope; malformed messages are deleted immediately
//     (poison-pill prevention — they would never succeed on retry).
//   - Call handler; on success the message is deleted from the queue.
//   - On handler error the message is left in-flight so SQS visibility timeout
//     and the queue's redrive policy handle retries and DLQ routing.
func (s *Subscriber) Subscribe(ctx context.Context, handler func(ctx context.Context, event Event) error) error {
	for {
		if ctx.Err() != nil {
			return ctx.Err()
		}

		msgs, err := s.sqs.Receive(ctx, s.queueURL, s.maxBatch)
		if err != nil {
			if ctx.Err() != nil {
				return ctx.Err()
			}
			logger.Error("events: receive failed", err)
			continue
		}

		for _, msg := range msgs {
			var evt Event
			if err := json.Unmarshal([]byte(msg.Body), &evt); err != nil {
				logger.Error(fmt.Sprintf("events: malformed message id=%s, deleting", msg.ID), err)
				_ = s.sqs.Delete(ctx, s.queueURL, msg.ReceiptHandle)
				continue
			}

			if err := handler(ctx, evt); err != nil {
				logger.Error(fmt.Sprintf("events: handler failed event=%s id=%s", evt.Type, evt.ID), err)
				// leave in-flight for SQS redrive / DLQ
				continue
			}

			if err := s.sqs.Delete(ctx, s.queueURL, msg.ReceiptHandle); err != nil {
				logger.Error(fmt.Sprintf("events: delete failed event=%s id=%s", evt.Type, evt.ID), err)
			}
		}
	}
}
