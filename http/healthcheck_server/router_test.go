package healthcheck_server_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/valyala/fasthttp"

	"github.com/pixality-inc/golang-core/env"
	"github.com/pixality-inc/golang-core/http/about"
	"github.com/pixality-inc/golang-core/http/healthcheck"
	"github.com/pixality-inc/golang-core/http/healthcheck_server"
)

type okService struct{}

func (s okService) IsOK() bool {
	return true
}

func TestNewRouter(t *testing.T) {
	t.Parallel()

	healthHandler := healthcheck.NewDefaultHandler(t.Context(), time.Hour, okService{})
	aboutHandler := about.NewHandler(env.New("test", "", "", "", "", "", time.Now()))

	handler := healthcheck_server.NewRouter(healthHandler, aboutHandler).Handle()

	for _, path := range []string{"/healthcheck", "/healthcheck/readiness", "/about"} {
		var ctx fasthttp.RequestCtx

		ctx.Request.SetRequestURI(path)
		ctx.Request.Header.SetMethod(fasthttp.MethodGet)

		handler(&ctx)

		assert.Equal(t, fasthttp.StatusOK, ctx.Response.StatusCode(), path)
	}
}
