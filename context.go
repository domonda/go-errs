package errs

import (
	"context"
	"errors"
)

// IsContextCanceled checks if the context Done channel is closed
// and if the context error unwraps to context.Canceled.
func IsContextCanceled(ctx context.Context) bool {
	select {
	case <-ctx.Done():
		return errors.Is(ctx.Err(), context.Canceled)
	default:
		return false
	}
}

// IsContextDeadlineExceeded checks if the context Done channel is closed
// and if the context error unwraps to context.DeadlineExceeded.
func IsContextDeadlineExceeded(ctx context.Context) bool {
	select {
	case <-ctx.Done():
		return errors.Is(ctx.Err(), context.DeadlineExceeded)
	default:
		return false
	}
}

// IsContextDone checks if the context Done channel is closed.
func IsContextDone(ctx context.Context) bool {
	select {
	case <-ctx.Done():
		return true
	default:
		return false
	}
}
