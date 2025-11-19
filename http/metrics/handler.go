package metrics

import (
	"github.com/pixality-inc/golang-core/metrics"

	"github.com/prometheus/common/expfmt"
	"github.com/valyala/fasthttp"
)

type Handler struct {
	manager metrics.Manager
}

func NewHandler(manager metrics.Manager) *Handler {
	return &Handler{
		manager: manager,
	}
}

func (h *Handler) GetMetrics(ctx *fasthttp.RequestCtx) {
	metricFamilies, err := h.manager.Gather()
	if err != nil {
		ctx.SetStatusCode(fasthttp.StatusInternalServerError)
		ctx.SetBodyString("Error gathering metrics")

		return
	}

	ctx.SetContentType("text/plain; version=0.0.4; charset=utf-8")
	ctx.SetStatusCode(fasthttp.StatusOK)

	encoder := expfmt.NewEncoder(ctx, expfmt.NewFormat(expfmt.TypeTextPlain))

	for _, mf := range metricFamilies {
		if err := encoder.Encode(mf); err != nil {
			// If encoding fails, we can't do much at this point
			// as headers have already been sent
			return
		}
	}
}
