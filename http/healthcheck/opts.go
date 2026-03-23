package healthcheck

import (
	"time"

	"github.com/pixality-inc/golang-core/logger"
)

type Options struct {
	ReCheckAfter time.Duration
	Logger       logger.Logger
}
