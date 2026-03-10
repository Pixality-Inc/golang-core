package kafka

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/twmb/franz-go/pkg/kerr"
)

var (
	errSomethingBroke = errors.New("something broke")
	errWrap           = errors.New("wrap")
)

func TestShouldIgnoreErrorForCircuitBreaker(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		err    error
		ignore bool
	}{
		{name: "nil error", err: nil, ignore: false},
		{name: "context canceled", err: context.Canceled, ignore: true},
		{name: "SASL auth failed", err: kerr.SaslAuthenticationFailed, ignore: true},
		{name: "topic authorization failed", err: kerr.TopicAuthorizationFailed, ignore: true},
		{name: "group authorization failed", err: kerr.GroupAuthorizationFailed, ignore: true},
		{name: "cluster authorization failed", err: kerr.ClusterAuthorizationFailed, ignore: true},
		{name: "random error", err: errSomethingBroke, ignore: false},
		{name: "wrapped context canceled", err: errors.Join(errWrap, context.Canceled), ignore: true},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			require.Equal(t, testCase.ignore, ShouldIgnoreErrorForCircuitBreaker(testCase.err))
		})
	}
}
