package readwriter

import "context"

type Option func(*Options)

type Options struct {
	Location string
	Schema   map[string]struct {
		Index int
		Type  string
	}
	Context context.Context
}

func WithLocation(loc string) Option {
	return func(o *Options) {
		o.Location = loc
	}
}

func WithSchema(schema map[string]struct {
	Index int
	Type  string
}) Option {
	return func(o *Options) {
		o.Schema = schema
	}
}

func NewOptions(opts ...Option) Options {
	options := Options{
		Context: context.Background(),
	}

	for _, fn := range opts {
		fn(&options)
	}

	return options
}
