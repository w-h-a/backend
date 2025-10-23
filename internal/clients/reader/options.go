package reader

import "context"

type ReadOneOption func(*ReadOneOptions)

type ReadOneOptions struct {
	Context context.Context
}

func NewReadOneOptions(opts ...ReadOneOption) ReadOneOptions {
	options := ReadOneOptions{
		Context: context.Background(),
	}

	for _, fn := range opts {
		fn(&options)
	}

	return options
}

type ListOption func(*ListOptions)

type ListOptions struct {
	SortBy  string
	Context context.Context
}

func WithSortBy(sortBy string) ListOption {
	return func(lo *ListOptions) {
		lo.SortBy = sortBy
	}
}

func NewListOptions(opts ...ListOption) ListOptions {
	options := ListOptions{
		Context: context.Background(),
	}

	for _, fn := range opts {
		fn(&options)
	}

	return options
}
