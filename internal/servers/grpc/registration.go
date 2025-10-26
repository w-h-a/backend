package grpc

import "google.golang.org/grpc"

type GrpcServiceRegistration struct {
	Desc *grpc.ServiceDesc
	Impl any
}
