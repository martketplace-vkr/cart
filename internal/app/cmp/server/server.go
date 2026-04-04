package server

import (
	"context"
	"net"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"

	"github.com/martketplace-vkr/cart/pkg/api/grpc/v1/client"
	cartorder "github.com/martketplace-vkr/cart/pkg/api/grpc/v1/order"
	grpcServer "github.com/martketplace-vkr/pkg/server/grpc"
)

const (
	cmpName = "GRPC server"
)

type Server struct {
	cfg        grpcServer.Config
	grpcServer *grpc.Server
	client     client.CartClientServiceServer
	order      cartorder.CartOrderServiceServer
}

func New(
	cfg grpcServer.Config,
	client client.CartClientServiceServer,
	order cartorder.CartOrderServiceServer,
) *Server {
	return &Server{
		cfg:    cfg,
		client: client,
		order:  order,
	}
}

func (s *Server) Start(ctx context.Context) (err error) {
	server, err := grpcServer.New(
		ctx,
		s.cfg,
		nil,
	)
	if err != nil {
		return err
	}

	s.grpcServer = server.Grpc
	reflection.Register(s.grpcServer)

	client.RegisterCartClientServiceServer(s.grpcServer, s.client)
	cartorder.RegisterCartOrderServiceServer(s.grpcServer, s.order)

	listener, err := net.Listen("tcp", s.cfg.Host)
	if err != nil {
		return err
	}
	errCh := make(chan error)

	go func() {
		if err := s.grpcServer.Serve(listener); err != nil {
			errCh <- err
		}
	}()
	select {
	case err := <-errCh:
		return err
	case <-time.After(s.cfg.StartTimeout.Duration):
		return nil
	}
}

func (s *Server) Stop(_ context.Context) error {
	stopCh := make(chan any)
	go func() {
		s.grpcServer.GracefulStop()
		stopCh <- nil
	}()
	select {
	case <-time.After(s.cfg.StopTimeout.Duration):
		return nil
	case <-stopCh:
		return nil
	}
}

func (c *Server) GetName() string {
	return cmpName
}

func (c *Server) GetShutdownDelay() time.Duration {
	return time.Second
}

func (c *Server) GetStartTimeout() time.Duration {
	return 5 * time.Second
}

func (c *Server) GetStopTimeout() time.Duration {
	return 5 * time.Second
}
