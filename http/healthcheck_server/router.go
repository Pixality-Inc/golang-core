package healthcheck_server

import (
	"github.com/pixality-inc/golang-core/http"
	"github.com/pixality-inc/golang-core/http/about"
	"github.com/pixality-inc/golang-core/http/healthcheck"
)

func NewRouter(
	healthHandler *healthcheck.Handler,
	aboutHandler *about.Handler,
) http.Router {
	router := http.NewRouter()

	router.GET("/healthcheck", healthHandler.GetReadiness)
	router.GET("/healthcheck/readiness", healthHandler.GetReadiness)
	router.GET("/about", aboutHandler.Get)

	return router
}
