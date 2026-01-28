package client

import (
	"context"
	"errors"
	"math/rand"
	"net"
	"time"

	"github.com/1password/onepassword-sdk-go"
)

const (
	defaultRetryAttempts = 5
	defaultRetryDelay    = 200 * time.Millisecond
	defaultRetryMaxDelay = 5 * time.Second
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

func withRetry[T any](ctx context.Context, fn func(context.Context) (T, error)) (T, error) {
	var zero T
	delay := defaultRetryDelay

	for attempt := 1; attempt <= defaultRetryAttempts; attempt++ {
		result, err := fn(ctx)
		if err == nil {
			return result, nil
		}

		if !isRetryableError(err) || attempt == defaultRetryAttempts {
			return zero, err
		}

		wait := addJitter(delay)
		if err := sleepWithContext(ctx, wait); err != nil {
			return zero, err
		}

		delay *= 2
		if delay > defaultRetryMaxDelay {
			delay = defaultRetryMaxDelay
		}
	}

	return zero, nil
}

func isRetryableError(err error) bool {
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return false
	}

	var rateLimitErr *onepassword.RateLimitExceededError
	if errors.As(err, &rateLimitErr) {
		return true
	}

	var netErr net.Error
	if errors.As(err, &netErr) {
		return netErr.Timeout() || netErr.Temporary()
	}

	return false
}

func addJitter(base time.Duration) time.Duration {
	if base <= 0 {
		return base
	}

	maxJitter := base / 2
	if maxJitter <= 0 {
		return base
	}

	jitter := time.Duration(rand.Int63n(int64(maxJitter)))
	return base + jitter
}

func sleepWithContext(ctx context.Context, wait time.Duration) error {
	timer := time.NewTimer(wait)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}
