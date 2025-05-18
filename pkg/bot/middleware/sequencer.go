package middleware

import (
	"context"
	"errors"
	"fmt"
	"sync"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// WithRequestSequencer creates middleware that ensures requests for the same user are processed
// sequentially in the order they were received. If there are already active requests for a user,
// new requests will wait until previous ones finish or be canceled if the request context is canceled.
// Returns a Middleware that enforces the sequential processing policy.
func WithRequestSequencer() Middleware {
	// Map to store request queues for each user
	var (
		userQueues = make(map[int64]chan struct{})
		mu         sync.Mutex
	)

	return func(next Handler) Handler {
		return HandlerFunc(func(ctx context.Context, message *tgbotapi.Message) (tgbotapi.MessageConfig, error) {
			if message == nil || message.From == nil {
				return tgbotapi.MessageConfig{}, errors.New("message or user is nil")
			}

			userID := message.From.ID

			// Get or create a queue for this user
			mu.Lock()
			queue, exists := userQueues[userID]
			if !exists {
				// First request for this user, create a channel and signal it's ready to process
				queue = make(chan struct{}, 1)
				queue <- struct{}{} // Signal that it's ready to process
				userQueues[userID] = queue
			}
			mu.Unlock()

			// Try to acquire the lock or wait for context cancellation
			select {
			case <-queue: // Wait for our turn
				// We got the lock, ensure we release it when done
				defer func() {
					mu.Lock()
					// Check if the queue is still in the map
					if q, ok := userQueues[userID]; ok && q == queue {
						// Signal the next request to proceed
						select {
						case queue <- struct{}{}:
							// Lock released, keep the queue
						default:
							// Queue is full or closed, this shouldn't happen
							// If this is the last request, remove the queue from the map
							delete(userQueues, userID)
						}
					}
					mu.Unlock()
				}()

				// Process the message
				resp, err := next.Handle(ctx, message)
				return resp, err
			case <-ctx.Done():
				// Context was cancelled while waiting for our turn
				return tgbotapi.MessageConfig{}, fmt.Errorf("context cancelled while waiting for user's previous requests to complete: %w", ctx.Err())
			}
		})
	}
}
