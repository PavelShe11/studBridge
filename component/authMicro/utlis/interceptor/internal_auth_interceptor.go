package interceptor

import (
	"context"
	"github.com/PavelShe11/studbridge/common/logger"

	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

const (
	// internalAPIKeyHeader is the metadata key for internal microservice API key authentication
	internalAPIKeyHeader = "x-internal-api-key"
)

// UnaryClientInternalAuthInterceptor creates a unary interceptor that injects API keys for internal microservice calls
func UnaryClientInternalAuthInterceptor(apiKey string, logger logger.Logger) grpc.UnaryClientInterceptor {
	return func(
		ctx context.Context,
		method string,
		req, reply interface{},
		cc *grpc.ClientConn,
		invoker grpc.UnaryInvoker,
		opts ...grpc.CallOption,
	) error {
		// Create new context with internal API key metadata
		ctx = metadata.AppendToOutgoingContext(ctx, internalAPIKeyHeader, apiKey)

		logger.Infof("Added internal API key to gRPC request: %s", method)

		// Invoke the remote method with the modified context
		return invoker(ctx, method, req, reply, cc, opts...)
	}
}
