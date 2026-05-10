package tokenizer

import (
	"context"
	"io"

	"github.com/pixality-inc/golang-core/iterator"
)

type Tokenizer interface {
	Tokenize(ctx context.Context, source io.Reader) (iterator.Iterator[Token], error)
}

type Impl struct {
	options *Options
}

func New(options ...Option) Tokenizer {
	tokenizerOptions := NewDefaultOptions()

	for _, option := range options {
		option(tokenizerOptions)
	}

	return &Impl{
		options: tokenizerOptions,
	}
}

func (t *Impl) Tokenize(ctx context.Context, source io.Reader) (iterator.Iterator[Token], error) {
	return NewState(source), nil
}
