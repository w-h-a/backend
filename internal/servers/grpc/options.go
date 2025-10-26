package grpc

import (
	"context"

	"github.com/w-h-a/backend/internal/servers"
	"google.golang.org/grpc"
)

type unaryInterceptorKey struct{}

func WithUnaryInterceptors(unaries ...grpc.UnaryServerInterceptor) servers.Option {
	return func(o *servers.Options) {
		o.Context = context.WithValue(o.Context, unaryInterceptorKey{}, unaries)
	}
}

func getUnaryInterceptorsFromCtx(ctx context.Context) ([]grpc.UnaryServerInterceptor, bool) {
	unaries, ok := ctx.Value(unaryInterceptorKey{}).([]grpc.UnaryServerInterceptor)
	return unaries, ok
}

type streamInterceptorKey struct{}

func WithStreamInterceptors(streamies ...grpc.StreamServerInterceptor) servers.Option {
	return func(o *servers.Options) {
		o.Context = context.WithValue(o.Context, streamInterceptorKey{}, streamies)
	}
}

func getStreamInterceptorsFromCtx(ctx context.Context) ([]grpc.StreamServerInterceptor, bool) {
	streamies, ok := ctx.Value(streamInterceptorKey{}).([]grpc.StreamServerInterceptor)
	return streamies, ok
}
