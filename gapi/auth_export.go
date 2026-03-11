package gapi

import (
	"context"

	"google.golang.org/grpc"
)

// AuthInterceptor returns the server's unary auth interceptor as a
// standalone grpc.UnaryServerInterceptor, ready to be chained by the caller.
//
// Usage in main.go:
//
//	grpc.ChainUnaryInterceptor(
//	    logger.UnaryServerInterceptor(opts),
//	    gapiServer.AuthInterceptor(),
//	)
func (server *Server) AuthInterceptor() grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req any,
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (any, error) {
		return server.authInterceptor(ctx, req, info, handler)
	}
}