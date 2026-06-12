package http

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/valyala/fasthttp"
)

func TestGetAuthorizationBearerToken(t *testing.T) {
	t.Parallel()

	var ctx fasthttp.RequestCtx

	ctx.Request.Header.Set(fasthttp.HeaderAuthorization, "Bearer secret-token")

	token, err := GetAuthorizationBearerToken(&ctx, false)
	require.NoError(t, err)
	assert.Equal(t, "secret-token", token)
}

func TestGetAuthorizationBearerTokenMissing(t *testing.T) {
	t.Parallel()

	var ctx fasthttp.RequestCtx

	_, err := GetAuthorizationBearerToken(&ctx, false)
	require.ErrorIs(t, err, ErrNoHeader)

	token, err := GetAuthorizationBearerToken(&ctx, true)
	require.NoError(t, err)
	assert.Empty(t, token)
}

func TestGetAuthorizationBearerTokenStoplightPlaceholder(t *testing.T) {
	t.Parallel()

	var ctx fasthttp.RequestCtx

	ctx.Request.Header.Set(fasthttp.HeaderAuthorization, "Bearer 123")

	_, err := GetAuthorizationBearerToken(&ctx, false)
	require.ErrorIs(t, err, ErrNoHeader)

	token, err := GetAuthorizationBearerToken(&ctx, true)
	require.NoError(t, err)
	assert.Empty(t, token)
}

func TestGetAuthorizationBearerTokenMalformed(t *testing.T) {
	t.Parallel()

	var ctx fasthttp.RequestCtx

	ctx.Request.Header.Set(fasthttp.HeaderAuthorization, "Basic dXNlcjpwYXNz")

	_, err := GetAuthorizationBearerToken(&ctx, false)
	require.ErrorIs(t, err, ErrHeaderIsMalformed)
}

func TestParseBool(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		value    string
		expected bool
	}{
		{"true", true},
		{"1", true},
		{"yes", true},
		{"TRUE", true},
		{"Yes", true},
		{"false", false},
		{"0", false},
		{"no", false},
		{"FALSE", false},
	}

	for _, testCase := range testCases {
		value, err := ParseBool(testCase.value)
		require.NoError(t, err)
		assert.Equal(t, testCase.expected, value, testCase.value)
	}

	_, err := ParseBool("maybe")
	require.ErrorIs(t, err, ErrUnknownValueForBool)
}

func TestParseUint64(t *testing.T) {
	t.Parallel()

	value, err := ParseUint64("42")
	require.NoError(t, err)
	assert.Equal(t, uint64(42), value)

	_, err = ParseUint64("-1")
	require.Error(t, err)

	_, err = ParseUint64("abc")
	require.Error(t, err)
}

func TestParseUUID(t *testing.T) {
	t.Parallel()

	id, err := ParseUUID("c2c524a3-9617-4d2c-a826-77d97acb2604")
	require.NoError(t, err)
	assert.Equal(t, "c2c524a3-9617-4d2c-a826-77d97acb2604", id.String())

	_, err = ParseUUID("not-a-uuid")
	require.Error(t, err)
}

func TestParseUnixTime(t *testing.T) {
	t.Parallel()

	parsed, err := ParseUnixTime("1700000000")
	require.NoError(t, err)
	assert.Equal(t, time.Unix(1700000000, 0), parsed)

	_, err = ParseUnixTime("not-a-number")
	require.Error(t, err)
}
