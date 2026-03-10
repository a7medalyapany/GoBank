package gapi

import (
	"context"
	"strings"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// authPayloadKey is the context key under which the verified token Payload is stored.
// Using a typed key avoids collisions with other context values.
type contextKey string

const authPayloadKey contextKey = "authorization_payload"

// publicRoutes lists gRPC full method names that do NOT require authentication.
// All other methods are protected by the auth interceptor.
var publicRoutes = map[string]bool{
	"/pb.GoBank/CreateUser":        true,
	"/pb.GoBank/LoginUser":         true,
	"/pb.GoBank/RenewAccessToken":  true,
}

// authInterceptor is a gRPC UnaryServerInterceptor that validates Bearer tokens.
// It skips validation for routes listed in publicRoutes.
// On success it injects the *token.Payload into the request context.
func (server *Server) authInterceptor(
	ctx context.Context,
	req any,
	info *grpc.UnaryServerInfo,
	handler grpc.UnaryHandler,
) (any, error) {
	// Skip auth for public routes
	if publicRoutes[info.FullMethod] {
		return handler(ctx, req)
	}

	// Extract metadata from the incoming context
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return nil, status.Errorf(codes.Unauthenticated, "missing request metadata")
	}

	// gRPC metadata keys are always lowercase
	values := md.Get("authorization")
	if len(values) == 0 {
		return nil, status.Errorf(codes.Unauthenticated, "authorization header is not provided")
	}

	fields := strings.Fields(values[0])
	if len(fields) < 2 {
		return nil, status.Errorf(codes.Unauthenticated, "invalid authorization header format")
	}

	if strings.ToLower(fields[0]) != "bearer" {
		return nil, status.Errorf(codes.Unauthenticated, "unsupported authorization type: %s", fields[0])
	}

	payload, err := server.tokenMaker.VerifyToken(fields[1])
	if err != nil {
		return nil, status.Errorf(codes.Unauthenticated, "invalid access token: %v", err)
	}

	// Inject payload into context for downstream handlers
	ctx = context.WithValue(ctx, authPayloadKey, payload)
	return handler(ctx, req)
}
