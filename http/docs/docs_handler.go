package docs

import (
	"strings"

	"github.com/pixality-inc/golang-core/http"
	"github.com/pixality-inc/golang-core/logger"

	"github.com/valyala/fasthttp"
)

type Handler struct {
	log         logger.Loggable
	docsDir     string
	fs          *fasthttp.FS
	fileHandler fasthttp.RequestHandler
}

func NewHandler(docsDir string, skipCache bool) *Handler {
	fileSystem := &fasthttp.FS{
		Root: docsDir,
		IndexNames: []string{
			"index.html",
		},
		Compress:           true,
		GenerateIndexPages: false,
		SkipCache:          skipCache,
	}

	return &Handler{
		log: logger.NewLoggableImplWithServiceAndFields(
			"handler",
			logger.Fields{
				"name": "docs",
			},
		),
		docsDir:     docsDir,
		fs:          fileSystem,
		fileHandler: fileSystem.NewRequestHandler(),
	}
}

func (h *Handler) Handle(ctx *fasthttp.RequestCtx) {
	path := string(ctx.Path())

	if newUri, found := strings.CutPrefix(path, "/docs"); found {
		ctx.URI().SetPath(newUri)

		h.fileHandler(ctx)

		return
	}

	http.NotFound(ctx, nil)
}
