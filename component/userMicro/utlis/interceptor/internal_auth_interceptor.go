package interceptor

import (
	"context"
	"userMicro/utlis/logger"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

const (
	// internalAPIKeyHeader is the metadata key for internal microservice API key authentication
	internalAPIKeyHeader = "x-internal-api-key"
)

// UnaryServerInternalAuthInterceptor creates a unary interceptor that validates API keys for internal microservice requests
func UnaryServerInternalAuthInterceptor(expectedAPIKey string, logger logger.Logger) grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req interface{},
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (interface{}, error) {
		// Extract metadata from context
		md, ok := metadata.FromIncomingContext(ctx)
		if !ok {
			logger.Warnf("Missing metadata in internal gRPC request to %s", info.FullMethod)
			return nil, status.Errorf(codes.Unauthenticated, "missing metadata")
		}

		// Get API key from metadata
		apiKeys := md.Get(internalAPIKeyHeader)
		if len(apiKeys) == 0 {
			logger.Warnf("Missing internal API key in gRPC request to %s", info.FullMethod)
			return nil, status.Errorf(codes.Unauthenticated, "missing API key")
		}

		// Validate API key
		if apiKeys[0] != expectedAPIKey {
			logger.Errorf("Invalid internal API key in gRPC request to %s", info.FullMethod)
			return nil, status.Errorf(codes.Unauthenticated, "invalid API key")
		}

		// API key is valid, proceed with the request
		logger.Infof("Authenticated internal gRPC request to %s", info.FullMethod)
		return handler(ctx, req)
	}
}
