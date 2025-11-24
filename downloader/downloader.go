package downloader

import (
	"context"

	http2 "github.com/pixality-inc/golang-core/http_client"
	"github.com/pixality-inc/golang-core/logger"
)

//go:generate mockgen -destination mocks/downloader_gen.go -source downloader.go
type Downloader interface {
	Download(ctx context.Context, url string) ([]byte, error)
}

type Impl struct {
	log  logger.Loggable
	http http2.Client
}

func NewDownloader(config http2.Config) (Downloader, error) {
	log := logger.NewLoggableImplWithService("downloader")

	httpClient, err := http2.NewClientImpl(log, config)
	if err != nil {
		return nil, err
	}

	return &Impl{
		log:  log,
		http: httpClient,
	}, nil
}

func (c *Impl) Download(ctx context.Context, url string) ([]byte, error) {
	log := c.log.GetLogger(ctx)

	log.Infof("Downloading from '%s'", url)

	response, err := c.http.Get(ctx, url)
	if err != nil {
		return nil, err
	}

	return response.GetBody(), nil
}
