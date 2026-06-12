package env_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/pixality-inc/golang-core/env"
)

func TestNew(t *testing.T) {
	t.Parallel()

	startedAt := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)

	appEnv := env.New(
		"staging",
		"pipeline-7",
		"v2.0.0",
		"develop",
		"0123456789abcdef",
		"0123456",
		startedAt,
	)

	assert.Equal(t, "staging", appEnv.EnvName())
	assert.Equal(t, "pipeline-7", appEnv.CiPipelineId())
	assert.Equal(t, "v2.0.0", appEnv.GitTag())
	assert.Equal(t, "develop", appEnv.GitBranch())
	assert.Equal(t, "0123456789abcdef", appEnv.GitCommit())
	assert.Equal(t, "0123456", appEnv.GitCommitShort())
	assert.Equal(t, startedAt, appEnv.StartedAt())
}

func TestNewEmptyValues(t *testing.T) {
	t.Parallel()

	appEnv := env.New("", "", "", "", "", "", time.Time{})

	assert.Empty(t, appEnv.EnvName())
	assert.Empty(t, appEnv.CiPipelineId())
	assert.Empty(t, appEnv.GitTag())
	assert.Empty(t, appEnv.GitBranch())
	assert.Empty(t, appEnv.GitCommit())
	assert.Empty(t, appEnv.GitCommitShort())
	assert.True(t, appEnv.StartedAt().IsZero())
}
