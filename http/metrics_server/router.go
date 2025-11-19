package metrics_server

import (
	"github.com/pixality-inc/golang-core/http"
	"github.com/pixality-inc/golang-core/http/metrics"
)

func NewRouter(
	metricsHandler *metrics.Handler,
) http.Router {
	router := http.NewRouter()

	router.GET("/metrics", metricsHandler.GetMetrics)

	return router
}
