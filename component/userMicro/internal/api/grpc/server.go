package proto

import (
	"net"
	"userMicro/internal/config"
	"userMicro/utlis/logger"

	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

type AccountService struct {
	address string
	server  *grpc.Server
	logger  logger.Logger
}

func NewGRPCServer(config config.GRPCConfig, logger logger.Logger) *AccountService {
	return &AccountService{
		address: config.ServerAddr,
		logger:  logger,
	}
}

func (s *AccountService) Start() error {
	lis, err := net.Listen("tcp", s.address)
	if err != nil {
		return err
	}

	server := grpc.NewServer()
	reflection.Register(server)

	s.server = server

	s.logger.Info("grpc server listening on " + s.address)
	return server.Serve(lis)
}

func (s *AccountService) Stop() {
	if s.server == nil {
		s.logger.Info("Gracefully stopping gRPC server")
		s.server.GracefulStop()
	}
}
