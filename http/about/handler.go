package about

import (
	"time"

	"github.com/pixality-inc/golang-core/clock"
	"github.com/pixality-inc/golang-core/env"
	"github.com/pixality-inc/golang-core/json"

	"github.com/valyala/fasthttp"
)

const TimeFormat = time.RFC3339

type (
	ResponseEnv struct {
		Name string `json:"name"`
	}

	ResponseUptime struct {
		Now           string  `json:"now"`
		StartedAt     string  `json:"started_at"`
		UptimeSeconds float64 `json:"uptime_seconds"`
	}

	ResponseCi struct {
		PipelineId string `json:"pipeline_id"`
	}

	ResponseGit struct {
		Tag         string `json:"tag"`
		Branch      string `json:"branch"`
		CommitShort string `json:"commit_short"`
		Commit      string `json:"commit"`
	}

	Response struct {
		Env    ResponseEnv    `json:"env"`
		Uptime ResponseUptime `json:"uptime"`
		Ci     ResponseCi     `json:"ci"`
		Git    ResponseGit    `json:"git"`
	}
)

type Handler struct {
	appEnv env.AppEnv
}

func NewHandler(
	appEnv env.AppEnv,
) *Handler {
	return &Handler{
		appEnv: appEnv,
	}
}

func (h *Handler) Get(ctx *fasthttp.RequestCtx) {
	now := clock.GetClock(ctx).Now()

	response := Response{
		Env: ResponseEnv{
			Name: h.appEnv.EnvName(),
		},
		Uptime: ResponseUptime{
			Now:           now.Format(TimeFormat),
			StartedAt:     h.appEnv.StartedAt().Format(TimeFormat),
			UptimeSeconds: now.Sub(h.appEnv.StartedAt()).Seconds(),
		},
		Ci: ResponseCi{
			PipelineId: h.appEnv.CiPipelineId(),
		},
		Git: ResponseGit{
			Tag:         h.appEnv.GitTag(),
			Branch:      h.appEnv.GitBranch(),
			CommitShort: h.appEnv.GitCommitShort(),
			Commit:      h.appEnv.GitCommit(),
		},
	}

	responseBytes, err := json.Marshal(response)
	if err != nil {
		ctx.SetStatusCode(fasthttp.StatusInternalServerError)

		return
	}

	ctx.SetStatusCode(fasthttp.StatusOK)
	ctx.Response.Header.Set("Content-Type", "application/json")
	ctx.SetBody(responseBytes)
}
