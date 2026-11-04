package errs

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"
)

func TestIsContextDone(t *testing.T) {
	cancelledCtx, cancel := context.WithCancel(context.Background())
	cancel()
	uncancelledCtx, cancelLater := context.WithCancel(context.Background())

	tests := []struct {
		name string
		ctx  context.Context
		want bool
	}{
		{name: "background", ctx: context.Background(), want: false},
		{name: "uncancelledCtx", ctx: uncancelledCtx, want: false},
		{name: "cancelledCtx", ctx: cancelledCtx, want: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsContextDone(tt.ctx); got != tt.want {
				t.Errorf("IsContextDone() = %v, want %v", got, tt.want)
			}
		})
	}

	cancelLater() // to avoid lint warnings
}

func TestIsContextCanceled(t *testing.T) {
	cancelledCtx, cancel := context.WithCancel(context.Background())
	cancel()
	uncancelledCtx, cancelLater := context.WithCancel(context.Background())

	tests := []struct {
		name string
		ctx  context.Context
		want bool
	}{
		{name: "background", ctx: context.Background(), want: false},
		{name: "uncancelledCtx", ctx: uncancelledCtx, want: false},
		{name: "cancelledCtx", ctx: cancelledCtx, want: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsContextCanceled(tt.ctx); got != tt.want {
				t.Errorf("IsContextCanceled() = %v, want %v", got, tt.want)
			}
		})
	}

	cancelLater() // to avoid lint warnings
}

func TestIsContextDeadlineExceeded(t *testing.T) {
	exceededDeadlineCtx, exceededDeadlineCtxCancel := context.WithTimeout(context.Background(), 0)
	futureDeadlineCtx, futureDeadlineCtxCancel := context.WithTimeout(context.Background(), time.Hour*24)

	time.Sleep(time.Millisecond) // For exceededDeadlineCtx

	cancelledCtx, cancel := context.WithCancel(context.Background())
	cancel()
	uncancelledCtx, cancelLater := context.WithCancel(context.Background())

	tests := []struct {
		name string
		ctx  context.Context
		want bool
	}{
		{name: "background", ctx: context.Background(), want: false},
		{name: "exceededDeadlineCtx", ctx: exceededDeadlineCtx, want: true},
		{name: "futureDeadlineCtx", ctx: futureDeadlineCtx, want: false},
		{name: "uncancelledCtx", ctx: uncancelledCtx, want: false},
		{name: "cancelledCtx", ctx: cancelledCtx, want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsContextDeadlineExceeded(tt.ctx); got != tt.want {
				t.Errorf("IsContextDeadlineExceeded() = %v, want %v", got, tt.want)
			}
		})
	}

	exceededDeadlineCtxCancel() // to avoid lint warnings
	futureDeadlineCtxCancel()   // to avoid lint warnings
	cancelLater()               // to avoid lint warnings
}

func TestIsContextError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{name: "nil", err: nil, want: false},
		{name: "Canceled", err: context.Canceled, want: true},
		{name: "DeadlineExceeded", err: context.DeadlineExceeded, want: true},
		{name: "wrapped Canceled", err: fmt.Errorf("wrapped: %w", context.Canceled), want: true},
		{name: "wrapped DeadlineExceeded", err: Errorf("wrapped: %w", context.DeadlineExceeded), want: true},
		{name: "other", err: errors.New("other"), want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsContextError(tt.err); got != tt.want {
				t.Errorf("IsContextError() = %v, want %v", got, tt.want)
			}
		})
	}
}
