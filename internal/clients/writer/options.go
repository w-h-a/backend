package writer

import "context"

type WriteOption func(*WriteOptions)

type WriteOptions struct {
	Context context.Context
}

func NewWriteOptions(opts ...WriteOption) WriteOptions {
	options := WriteOptions{
		Context: context.Background(),
	}

	for _, fn := range opts {
		fn(&options)
	}

	return options
}

type UpdateOption func(*UpdateOptions)

type UpdateOptions struct {
	Context context.Context
}

func NewUpdateOptions(opts ...UpdateOption) UpdateOptions {
	options := UpdateOptions{
		Context: context.Background(),
	}

	for _, fn := range opts {
		fn(&options)
	}

	return options
}

type DeleteOption func(*DeleteOptions)

type DeleteOptions struct {
	Context context.Context
}

func NewDeleteOptions(opts ...DeleteOption) DeleteOptions {
	options := DeleteOptions{
		Context: context.Background(),
	}

	for _, fn := range opts {
		fn(&options)
	}

	return options
}
