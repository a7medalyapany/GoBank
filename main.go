package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"strings"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"

	"github.com/a7medalyapany/GoBank.git/api"
	db "github.com/a7medalyapany/GoBank.git/db/sqlc"
	"github.com/a7medalyapany/GoBank.git/gapi"
	"github.com/a7medalyapany/GoBank.git/pb"
	"github.com/a7medalyapany/GoBank.git/util"
	"github.com/jackc/pgx/v5/pgxpool"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/reflection"
	"google.golang.org/protobuf/encoding/protojson"
)

func main() {
	config, err := util.LoadConfig(".")
	if err != nil {
		panic(fmt.Sprintf("cannot load config: %v", err))
	}

	conn, err := pgxpool.New(context.Background(), config.DB_URL)
	if err != nil {
		panic(fmt.Sprintf("cannot connect to db: %v", err))
	}
	defer conn.Close()

	store := db.NewStore(conn)

	go runGatewayServer(store, config)
	runGRPCServer(store, config)
}

// runGinServer starts the Gin HTTP REST server.
// Uncomment in main() if you want to serve Gin alongside gRPC.
func runGinServer(store *db.Store, config util.Config) {
	server, err := api.NewServer(store, config)
	if err != nil {
		panic(fmt.Sprintf("cannot create Gin server: %v", err))
	}

	address := fmt.Sprintf("%s:%s", config.SERVER_ADDRESS, config.PORT)
	if err := server.Start(address); err != nil {
		panic(fmt.Sprintf("cannot start Gin server: %v", err))
	}
}

// runGRPCServer starts the gRPC server with the auth interceptor wired in.
func runGRPCServer(store *db.Store, config util.Config) {
	server, err := gapi.NewServer(store, config)
	if err != nil {
		panic(fmt.Sprintf("cannot create gRPC server: %v", err))
	}

	// GRPCServer() returns a *grpc.Server pre-configured with the auth interceptor
	grpcServer := server.GRPCServer()
	pb.RegisterGoBankServer(grpcServer, server)
	reflection.Register(grpcServer)

	address := fmt.Sprintf("%s:%s", config.SERVER_ADDRESS, config.GRPC_SERVER_PORT)
	listener, err := net.Listen("tcp", address)
	if err != nil {
		panic(fmt.Sprintf("cannot create gRPC listener: %v", err))
	}

	log.Printf("gRPC server listening at %s", listener.Addr().String())

	if err := grpcServer.Serve(listener); err != nil {
		panic(fmt.Sprintf("cannot serve gRPC: %v", err))
	}
}

// runGatewayServer starts the gRPC-Gateway HTTP server.
//
// Routes:
//
//	/swagger/          → custom branded Swagger UI
//	/swagger/doc.json  → raw embedded OpenAPI spec
//	/*                 → gRPC-Gateway (REST → gRPC translation)
func runGatewayServer(store *db.Store, config util.Config) {
	_, err := gapi.NewServer(store, config)
	if err != nil {
		panic(fmt.Sprintf("cannot create gateway server: %v", err))
	}

	jsonOption := runtime.WithMarshalerOption(runtime.MIMEWildcard, &runtime.JSONPb{
		MarshalOptions: protojson.MarshalOptions{
			UseProtoNames: true,
		},
		UnmarshalOptions: protojson.UnmarshalOptions{
			DiscardUnknown: true,
		},
	})

	grpcMux := runtime.NewServeMux(
		jsonOption,
		runtime.WithIncomingHeaderMatcher(func(key string) (string, bool) {
			if strings.ToLower(key) == "authorization" {
				return "authorization", true
			}
			return runtime.DefaultHeaderMatcher(key)
		}),
	)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// ← This routes HTTP → actual gRPC server (interceptor runs)
	// instead of RegisterGoBankHandlerServer which bypasses interceptor
	grpcEndpoint := fmt.Sprintf("%s:%s", config.SERVER_ADDRESS, config.GRPC_SERVER_PORT)
	err = pb.RegisterGoBankHandlerFromEndpoint(ctx, grpcMux, grpcEndpoint, []grpc.DialOption{
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	})
	if err != nil {
		panic(fmt.Sprintf("cannot register gateway handler: %v", err))
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/swagger/", gapi.SwaggerHandler)
	mux.Handle("/", grpcMux)

	httpAddress := fmt.Sprintf("%s:%s", config.SERVER_ADDRESS, config.PORT)
	listener, err := net.Listen("tcp", httpAddress)
	if err != nil {
		panic(fmt.Sprintf("cannot create HTTP listener: %v", err))
	}

	log.Printf("HTTP gateway listening at    http://%s", httpAddress)
	log.Printf("Swagger UI available at      http://%s/swagger/", httpAddress)
	log.Printf("OpenAPI spec available at    http://%s/swagger/doc.json", httpAddress)

	if err = http.Serve(listener, mux); err != nil {
		panic(fmt.Sprintf("cannot serve HTTP gateway: %v", err))
	}
}
