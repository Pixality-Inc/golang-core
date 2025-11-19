package http

import (
	"github.com/pixality-inc/golang-core/logger"

	"github.com/valyala/fasthttp"
)

func AddLogsExpanders(ctx *fasthttp.RequestCtx, logsExpanders ...logger.LogsExpander) {
	currentLogsExpanders, ok := ctx.UserValue(logger.LogsExpandersName).(logger.LogsExpanders)
	if !ok {
		currentLogsExpanders = logger.LogsExpanders{}
	}

	ctx.SetUserValue(logger.LogsExpandersName, append(currentLogsExpanders, logsExpanders...))
}
