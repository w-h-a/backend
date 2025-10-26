package grpc

import (
	"context"
	"errors"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/w-h-a/backend/internal/servers"
	"google.golang.org/grpc"
)

type grpcServer struct {
	options   servers.Options
	server    *grpc.Server
	errCh     chan error
	exit      chan struct{}
	isRunning bool
	mtx       sync.RWMutex
}

func (s *grpcServer) Handle(handler any) error {
	s.mtx.RLock()
	defer s.mtx.RUnlock()

	if s.isRunning {
		return errors.New("cannot set handler after server has started")
	}

	reg, ok := handler.(GrpcServiceRegistration)
	if !ok {
		return fmt.Errorf("invalid handler type: expected grpcServiceRegistration, got %T", handler)
	}

	if reg.Desc == nil || reg.Impl == nil {
		return errors.New("GrpcServiceRegistration requires both Desc and Impl")
	}

	s.server.RegisterService(reg.Desc, reg.Impl)

	return nil
}

func (s *grpcServer) Run(stop chan struct{}) error {
	s.mtx.RLock()
	if s.isRunning {
		s.mtx.RUnlock()
		return errors.New("server already running")
	}
	s.mtx.RUnlock()

	if err := s.Start(); err != nil {
		return fmt.Errorf("failed to start server: %w", err)
	}

	select {
	case err := <-s.errCh:
		stopCtx, stopCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer stopCancel()
		_ = s.stop(stopCtx)
		return err
	case <-stop:
		stopCtx, stopCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer stopCancel()
		return s.stop(stopCtx)
	}
}

func (s *grpcServer) Start() error {
	s.mtx.Lock()
	defer s.mtx.Unlock()

	if s.isRunning {
		return errors.New("server already started")
	}

	listener, err := net.Listen("tcp", s.options.Address)
	if err != nil {
		return err
	}

	s.options.Address = listener.Addr().String()

	s.exit = make(chan struct{})
	s.errCh = make(chan error, 1)

	go func() {
		if err := s.server.Serve(listener); err != nil {
			s.errCh <- fmt.Errorf("grpc server Serve error: %w", err)
		}
		close(s.exit)
	}()

	s.isRunning = true

	return nil
}

func (s *grpcServer) Stop() error {
	stopCtx, stopCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer stopCancel()
	return s.stop(stopCtx)
}

func (s *grpcServer) stop(ctx context.Context) error {
	s.mtx.Lock()

	if !s.isRunning {
		s.mtx.Unlock()
		return errors.New("server not running")
	}

	s.isRunning = false
	srv := s.server
	exit := s.exit

	s.mtx.Unlock()

	gracefulStopDone := make(chan struct{})
	go func() {
		srv.GracefulStop()
		close(gracefulStopDone)
	}()

	var stopErr error

	select {
	case <-gracefulStopDone:
	case <-ctx.Done():
		stopErr = ctx.Err()
		srv.Stop()
		<-gracefulStopDone
	}

	<-exit

	select {
	case err := <-s.errCh:
		return err
	default:
		return stopErr
	}
}

func NewServer(opts ...servers.Option) servers.Server {
	options := servers.NewOptions(opts...)

	// add interceptor 'middleware' and otel instrumentation here
	srv := grpc.NewServer()

	s := &grpcServer{
		options: options,
		server:  srv,
		mtx:     sync.RWMutex{},
	}

	return s
}
