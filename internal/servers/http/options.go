package http

import (
	"context"

	"github.com/w-h-a/backend/internal/servers"
)

type middlewareKey struct{}

func WithMiddleware(ms ...Middleware) servers.Option {
	return func(o *servers.Options) {
		o.Context = context.WithValue(o.Context, middlewareKey{}, ms)
	}
}

func getMiddlewareFromCtx(ctx context.Context) ([]Middleware, bool) {
	ms, ok := ctx.Value(middlewareKey{}).([]Middleware)
	return ms, ok
}
