package grpc

import (
	"net"
	"userMicro/internal/config"
	"userMicro/utlis/logger"

	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

type Service struct {
	address string
	Server  *grpc.Server
	logger  logger.Logger
}

func NewGRPCServer(config config.GRPCConfig, logger logger.Logger) *Service {
	return &Service{
		address: config.ServerAddr,
		logger:  logger,
	}
}

func (s *Service) Start() error {
	lis, err := net.Listen("tcp", s.address)
	if err != nil {
		return err
	}

	server := grpc.NewServer()
	reflection.Register(server)

	s.Server = server

	s.logger.Info("grpc Server listening on " + s.address)
	return server.Serve(lis)
}

func (s *Service) Stop() {
	if s.Server == nil {
		s.logger.Info("Gracefully stopping gRPC Server")
		s.Server.GracefulStop()
	}
}
