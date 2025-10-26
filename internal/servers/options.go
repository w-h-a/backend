package servers

import "context"

type Option func(*Options)

type Options struct {
	Address string
	Context context.Context
}

func WithAddress(addr string) Option {
	return func(o *Options) {
		o.Address = addr
	}
}

func NewOptions(opts ...Option) Options {
	options := Options{
		Address: ":0",
		Context: context.Background(),
	}

	for _, fn := range opts {
		fn(&options)
	}

	return options
}
