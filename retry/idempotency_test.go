package retry

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsIdempotentMethod(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		method   string
		expected bool
	}{
		{http.MethodGet, true},
		{http.MethodHead, true},
		{http.MethodPut, true},
		{http.MethodDelete, true},
		{http.MethodOptions, true},
		{http.MethodTrace, true},
		{http.MethodPost, false},
		{http.MethodPatch, false},
		{"get", true},
		{"post", false},
		{"", true},
		{"CONNECT", false},
		{"PROPFIND", false},
	}

	for _, tc := range testCases {
		t.Run(tc.method, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tc.expected, IsIdempotentMethod(tc.method))
		})
	}
}

func TestShouldRetryMethod_NilPolicy(t *testing.T) {
	t.Parallel()

	assert.True(t, ShouldRetryMethod(http.MethodGet, nil))
	assert.False(t, ShouldRetryMethod(http.MethodPost, nil))
}

func TestShouldRetryMethod_DefaultPolicy(t *testing.T) {
	t.Parallel()

	policy := NewPolicy()

	assert.False(t, policy.RetryNonIdempotent())
	assert.True(t, ShouldRetryMethod(http.MethodDelete, policy))
	assert.True(t, ShouldRetryMethod(http.MethodPut, policy))
	assert.False(t, ShouldRetryMethod(http.MethodPost, policy))
	assert.False(t, ShouldRetryMethod(http.MethodPatch, policy))
}

func TestShouldRetryMethod_RetryNonIdempotentEnabled(t *testing.T) {
	t.Parallel()

	policy := NewPolicy(WithRetryNonIdempotent(true))

	assert.True(t, policy.RetryNonIdempotent())
	assert.True(t, ShouldRetryMethod(http.MethodPost, policy))
	assert.True(t, ShouldRetryMethod(http.MethodPatch, policy))
	assert.True(t, ShouldRetryMethod(http.MethodGet, policy))
}

func TestRetryNonIdempotent_ConfigYaml(t *testing.T) {
	t.Parallel()

	cfg := &ConfigYaml{RetryNonIdempotentValue: true}
	assert.True(t, cfg.RetryNonIdempotent())

	cfg = &ConfigYaml{}
	assert.False(t, cfg.RetryNonIdempotent())
}
