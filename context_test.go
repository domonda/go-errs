package errs

import (
	"context"
	"testing"
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
