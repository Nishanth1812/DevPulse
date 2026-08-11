package ai

import (
	"context"
	"errors"
	"math/rand"
	"strings"
	"time"
)

const (
	maxAttempts = 3
	baseBackoff = 500 * time.Millisecond
	maxBackoff  = 5 * time.Second
)

// retryable reports whether an error is transient and worth retrying: rate
// limits, server errors, and timeouts. User cancellation is never retried.
func retryable(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, context.Canceled) {
		return false
	}

	msg := strings.ToLower(err.Error())
	for _, pattern := range []string{
		"rate limit", "rate_limit", "too many requests", "429",
		"500", "502", "503", "504",
		"timeout", "deadline exceeded", "unavailable", "temporary",
		"connection reset", "reset by peer",
	} {
		if strings.Contains(msg, pattern) {
			return true
		}
	}
	return false
}

// withRetry runs fn up to maxAttempts times, backing off exponentially with
// jitter between attempts, but only for retryable errors. The context is
// honoured between attempts.
func withRetry(ctx context.Context, fn func(ctx context.Context) (string, error)) (string, error) {
	var lastErr error

	for attempt := 1; attempt <= maxAttempts; attempt++ {
		result, err := fn(ctx)
		if err == nil {
			return result, nil
		}

		lastErr = err
		if !retryable(err) || attempt == maxAttempts {
			return result, err
		}

		delay := baseBackoff << (attempt - 1)
		if delay > maxBackoff {
			delay = maxBackoff
		}
		delay += time.Duration(rand.Int63n(int64(delay) / 2))

		timer := time.NewTimer(delay)
		select {
		case <-ctx.Done():
			timer.Stop()
			return result, ctx.Err()
		case <-timer.C:
		}
	}

	return "", lastErr
}
