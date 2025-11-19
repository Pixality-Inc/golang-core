package providers

import (
	"context"

	"github.com/pixality-inc/golang-core/storage"
)

type NoUrlProvider struct {
	prefix string
}

var NoUrlProviderImpl storage.UrlProvider = NewNoUrlProvider("")

func NewNoUrlProvider(prefix string) storage.UrlProvider {
	return &NoUrlProvider{
		prefix: prefix,
	}
}

func (p *NoUrlProvider) GetPublicUrl(ctx context.Context, path string) (string, error) {
	return p.prefix + path, nil
}
