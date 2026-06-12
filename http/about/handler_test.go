package about_test

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/valyala/fasthttp"

	"github.com/pixality-inc/golang-core/env"
	"github.com/pixality-inc/golang-core/http/about"
)

func TestHandlerGet(t *testing.T) {
	t.Parallel()

	startedAt := time.Now().Add(-time.Minute)
	appEnv := env.New(
		"production",
		"pipeline-42",
		"v1.2.3",
		"main",
		"abcdef1234567890",
		"abcdef1",
		startedAt,
	)

	handler := about.NewHandler(appEnv)

	var ctx fasthttp.RequestCtx

	handler.Get(&ctx)

	require.Equal(t, fasthttp.StatusOK, ctx.Response.StatusCode())
	require.Equal(t, "application/json", string(ctx.Response.Header.ContentType()))

	var response about.Response

	require.NoError(t, json.Unmarshal(ctx.Response.Body(), &response))

	assert.Equal(t, "production", response.Env.Name)
	assert.Equal(t, "pipeline-42", response.Ci.PipelineId)
	assert.Equal(t, "v1.2.3", response.Git.Tag)
	assert.Equal(t, "main", response.Git.Branch)
	assert.Equal(t, "abcdef1234567890", response.Git.Commit)
	assert.Equal(t, "abcdef1", response.Git.CommitShort)
	assert.Equal(t, startedAt.Format(about.TimeFormat), response.Uptime.StartedAt)
	assert.Positive(t, response.Uptime.UptimeSeconds)
	assert.NotEmpty(t, response.Uptime.Now)
}
