package tokenizer

type Options struct{}

func NewDefaultOptions() *Options {
	return &Options{}
}

type Option = func(options *Options)
