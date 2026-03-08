package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"

	"github.com/a7medalyapany/GoBank.git/api"
	db "github.com/a7medalyapany/GoBank.git/db/sqlc"
	"github.com/a7medalyapany/GoBank.git/gapi"
	"github.com/a7medalyapany/GoBank.git/pb"
	"github.com/a7medalyapany/GoBank.git/util"
	"github.com/jackc/pgx/v5/pgxpool"
	"google.golang.org/grpc"
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

	// comment this out once you fully migrate to gRPC
	// runGinServer(store, config)

	go runGatewayServer(store, config)
	runGRPCServer(store, config)
}

// runGinServer starts the HTTP REST server using Gin.
// Keep this around if you still want to serve REST alongside gRPC.
func runGinServer(store *db.Store, config util.Config,) {
	server, err := api.NewServer(store, config)
	if err != nil {
		panic(fmt.Sprintf("cannot create Gin server: %v", err))
	}

	address := fmt.Sprintf("%s:%s", config.SERVER_ADDRESS, config.PORT)
	if err := server.Start(address); err != nil {
		panic(fmt.Sprintf("cannot start Gin server: %v", err))
	}
}

// runGRPCServer starts the gRPC server.
func runGRPCServer(store *db.Store, config util.Config,) {
	server, err := gapi.NewServer(store, config)
	if err != nil {
		panic(fmt.Sprintf("cannot create gRPC server: %v", err))
	}

	grpcServer := grpc.NewServer()
	pb.RegisterGoBankServer(grpcServer, server)

	// reflection allows Evans and other tools to discover services at runtime
	reflection.Register(grpcServer)

	address := fmt.Sprintf("%s:%s", config.SERVER_ADDRESS, config.GRPC_SERVER_PORT)
	listener, err := net.Listen("tcp", address)
	if err != nil {
		panic(fmt.Sprintf("cannot create listener: %v", err))
	}

	log.Printf("gRPC server listening at %s", listener.Addr().String())

	if err := grpcServer.Serve(listener); err != nil {
		panic(fmt.Sprintf("cannot serve gRPC: %v", err))
	}
}

// runGatewayServer starts the gRPC-Gateway server to translate REST calls to gRPC.
func runGatewayServer(store *db.Store, config util.Config,) {
	server, err := gapi.NewServer(store, config)
	if err != nil {
		panic(fmt.Sprintf("cannot create gRPC server: %v", err))
	}

	jsonOption := runtime.WithMarshalerOption(runtime.MIMEWildcard, &runtime.JSONPb{
		MarshalOptions: protojson.MarshalOptions{
			UseProtoNames: true,
		},
		UnmarshalOptions: protojson.UnmarshalOptions{
			DiscardUnknown: true,
		},
	})

	grpcMux := runtime.NewServeMux(jsonOption)
	
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	err = pb.RegisterGoBankHandlerServer(ctx, grpcMux, server)
	if err != nil {
		panic(fmt.Sprintf("cannot register gRPC gateway: %v", err))
	}

	mux := http.NewServeMux()
	mux.Handle("/", grpcMux)

	httpAddress := fmt.Sprintf("%s:%s", config.SERVER_ADDRESS, config.PORT)
	listener, err := net.Listen("tcp", httpAddress)
	if err != nil {
		panic(fmt.Sprintf("cannot create listener: %v", err))
	}

	log.Printf("http gateway listening at %s", listener.Addr().String())

	if err := http.Serve(listener, mux); err != nil {
		panic(fmt.Sprintf("cannot serve http gateway: %v", err))
	}
}