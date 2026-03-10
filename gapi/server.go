package gapi

import (
	"fmt"

	db "github.com/a7medalyapany/GoBank.git/db/sqlc"
	"github.com/a7medalyapany/GoBank.git/pb"
	"github.com/a7medalyapany/GoBank.git/token"
	"github.com/a7medalyapany/GoBank.git/util"
	"google.golang.org/grpc"
)

// Server represents the gRPC server for the GoBank service.
type Server struct {
	pb.UnimplementedGoBankServer
	store      *db.Store
	config     util.Config
	tokenMaker token.Maker
}

// NewServer creates a new gRPC Server instance with a Paseto token maker.
func NewServer(store *db.Store, config util.Config) (*Server, error) {
	tokenMaker, err := token.NewPasetoMaker(config.TOKEN_SYMMETRIC_KEY)
	if err != nil {
		return nil, fmt.Errorf("cannot create token maker: %w", err)
	}

	return &Server{
		store:      store,
		config:     config,
		tokenMaker: tokenMaker,
	}, nil
}

// GRPCServer builds and returns a *grpc.Server with the auth interceptor wired in.
// Call this in main.go instead of constructing grpc.NewServer() manually.
//
// Usage in main.go:
//
//	grpcServer := gapiServer.GRPCServer()
//	pb.RegisterGoBankServer(grpcServer, gapiServer)
func (server *Server) GRPCServer() *grpc.Server {
	return grpc.NewServer(
		grpc.UnaryInterceptor(server.authInterceptor),
	)
}
