package grpc

import (
	"net"
	"userMicro/internal/config"
	"userMicro/utlis/interceptor"
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
	// Create internal auth interceptor for microservice-to-microservice authentication
	authInterceptor := interceptor.UnaryServerInternalAuthInterceptor(config.InternalAPIKey, logger)

	// Create server with interceptor
	server := grpc.NewServer(
		grpc.UnaryInterceptor(authInterceptor),
	)

	return &Service{
		address: config.ServerAddr,
		Server:  server,
		logger:  logger,
	}
}

func (s *Service) Start() error {
	lis, err := net.Listen("tcp", s.address)
	if err != nil {
		return err
	}

	reflection.Register(s.Server)

	s.logger.Info("grpcService Server listening on " + s.address)
	return s.Server.Serve(lis)
}

func (s *Service) Stop() {
	if s.Server != nil {
		s.logger.Info("Gracefully stopping gRPC Server")
		s.Server.GracefulStop()
	}
}
