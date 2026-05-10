package future

import "github.com/pixality-inc/golang-core/pool"

type Options struct {
	poolExecutor pool.PoolExecutor
}

func NewDefaultOptions() *Options {
	return &Options{
		poolExecutor: pool.Default,
	}
}

type Option func(options *Options)

func WithPoolExecutor(pe pool.PoolExecutor) Option {
	return func(options *Options) {
		options.poolExecutor = pe
	}
}
