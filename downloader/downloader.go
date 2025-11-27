package downloader

import (
	"context"

	http "github.com/pixality-inc/golang-core/http_client"
	"github.com/pixality-inc/golang-core/logger"
)

//go:generate mockgen -destination mocks/downloader_gen.go -source downloader.go
type Downloader interface {
	Download(ctx context.Context, url string) ([]byte, error)
}

type Impl struct {
	log        logger.Loggable
	httpClient http.Client
}

func NewDownloader(config http.Config) (Downloader, error) {
	log := logger.NewLoggableImplWithService("downloader")

	httpClient, err := http.NewClientImpl(log, config)
	if err != nil {
		return nil, err
	}

	return &Impl{
		log:        log,
		httpClient: httpClient,
	}, nil
}

func (c *Impl) Download(ctx context.Context, url string) ([]byte, error) {
	log := c.log.GetLogger(ctx)

	log.Infof("Downloading from '%s'", url)

	response, err := c.httpClient.Get(ctx, url)
	if err != nil {
		return nil, err
	}

	return response.GetBody(), nil
}
