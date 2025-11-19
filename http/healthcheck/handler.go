package healthcheck

import (
	"context"
	"sync"
	"time"

	"github.com/valyala/fasthttp"
)

type Handler struct {
	mx sync.RWMutex

	options Options
	service []Service

	ok bool
}

func NewDefaultHandler(ctx context.Context, reCheckDuration time.Duration, services ...Service) *Handler {
	return NewHandler(ctx, Options{
		ReCheckAfter: reCheckDuration,
	}, services...)
}

func NewHandler(ctx context.Context, opts Options, services ...Service) *Handler {
	handler := &Handler{
		options: opts,
		service: services,
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

	for i := range h.service {
		ok = ok && h.service[i].IsOK()

		if !ok {
			break
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
