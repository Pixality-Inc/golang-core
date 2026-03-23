package healthcheck

import (
	"context"
	"sync"
	"time"

	"github.com/valyala/fasthttp"
)

type Handler struct {
	mx sync.RWMutex

	options  Options
	services []Service

	ok bool
}

func NewDefaultHandler(ctx context.Context, reCheckDuration time.Duration, services ...Service) *Handler {
	return NewHandler(ctx, Options{
		ReCheckAfter: reCheckDuration,
	}, services...)
}

func NewHandler(ctx context.Context, opts Options, services ...Service) *Handler {
	handler := &Handler{
		options:  opts,
		services: services,
	}

	handler.performCheck()

	go handler.check(ctx, time.NewTicker(opts.ReCheckAfter))

	return handler
}

func (h *Handler) GetReadiness(ctx *fasthttp.RequestCtx) {
	h.mx.RLock()
	defer h.mx.RUnlock()

	if !h.ok {
		ctx.SetStatusCode(fasthttp.StatusServiceUnavailable)

		return
	}

	ctx.SetStatusCode(fasthttp.StatusOK)
}

func (h *Handler) performCheck() {
	ok := true

	for _, svc := range h.services {
		if !svc.IsOK() {
			ok = false

			if h.options.Logger != nil {
				name := "unknown"
				if ns, isNamed := svc.(NamedService); isNamed {
					name = ns.Name()
				}

				h.options.Logger.Errorf("healthcheck failed: %s", name)
			}
		}
	}

	h.mx.Lock()
	h.ok = ok
	h.mx.Unlock()
}

func (h *Handler) check(ctx context.Context, ticker *time.Ticker) {
	for {
		select {
		case <-ticker.C:
			h.performCheck()

		case <-ctx.Done():
			return
		}
	}
}
