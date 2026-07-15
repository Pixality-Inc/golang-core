package gcs

import (
	"context"
	"errors"
	"fmt"
	"net"
	"testing"

	"github.com/stretchr/testify/require"
	"golang.org/x/net/http2"
	"google.golang.org/api/googleapi"
)

var (
	errConnReset = errors.New("connection reset by peer")
	errBoom      = errors.New("boom")
)

func TestIsRetryableGcsErr(t *testing.T) {
	t.Parallel()

	// mirrors the observed failure: "stream error: stream ID 15; INTERNAL_ERROR; received from peer"
	streamErr := http2.StreamError{StreamID: 15, Code: http2.ErrCodeInternal}

	tests := []struct {
		name string
		err  error
		want bool
	}{
		{"nil", nil, false},
		{"http2 INTERNAL_ERROR stream reset", streamErr, true},
		{"wrapped http2 stream reset", fmt.Errorf("storage.Write(x): %w", streamErr), true},
		{"net error (connection reset)", &net.OpError{Op: "read", Err: errConnReset}, true},
		{"googleapi 500", &googleapi.Error{Code: 500}, true},
		{"googleapi 503", &googleapi.Error{Code: 503}, true},
		{"googleapi 429", &googleapi.Error{Code: 429}, true},
		{"googleapi 404 not found", &googleapi.Error{Code: 404}, false},
		{"googleapi 403 forbidden", &googleapi.Error{Code: 403}, false},
		{"context canceled", context.Canceled, false},
		{"context deadline exceeded", context.DeadlineExceeded, false},
		{"generic error", errBoom, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			require.Equal(t, tt.want, isRetryableGcsErr(tt.err))
		})
	}
}
