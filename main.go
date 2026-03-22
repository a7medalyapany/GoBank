package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"strings"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/hibiken/asynq"

	"github.com/a7medalyapany/GoBank.git/api"
	db "github.com/a7medalyapany/GoBank.git/db/sqlc"
	"github.com/a7medalyapany/GoBank.git/gapi"
	"github.com/a7medalyapany/GoBank.git/logger"
	"github.com/a7medalyapany/GoBank.git/mail"
	"github.com/a7medalyapany/GoBank.git/pb"
	"github.com/a7medalyapany/GoBank.git/util"
	"github.com/a7medalyapany/GoBank.git/worker"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
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

	logCfg := logger.DefaultConfig("go-bank", "1.0.0", config.ENVIRONMENT)
	if err := logger.InitGlobal(logCfg); err != nil {
		log.Fatalf("cannot init logger: %v", err)
	}
	defer logger.G().Sync() // nolint: errcheck

	l := logger.G()
	l.Info("starting GoBank",
		zap.String("port", config.PORT),
		zap.String("grpc_port", config.GRPC_SERVER_PORT),
	)

	conn, err := pgxpool.New(context.Background(), config.DB_URL)
	if err != nil {
		l.Fatal("cannot connect to db", zap.Error(err))
	}
	defer conn.Close()

	store := db.NewStore(conn)

	redisOpt := asynq.RedisClientOpt{
		Addr: config.REDIS_ADDRESS,
	}

	taskDistributor := worker.NewRedisTaskDistributor(redisOpt)
	
	go runGatewayServer(config)
	go runTaskProcessor(redisOpt, store, config)

	runGRPCServer(store, config, taskDistributor)
	// runGinServer(store, config) // kept for reference
}


func runTaskProcessor(redisOpt asynq.RedisClientOpt, store *db.Store, config util.Config) {
	l := logger.G()

	sender, err := mail.NewGmailSender(
		config.EMAIL_SENDER_NAME,
		config.EMAIL_SENDER_ADDRESS,
		config.EMAIL_SENDER_PASSWORD,
	)
	
	if err != nil {
		l.Fatal("cannot create mailer", zap.Error(err))
	}

	taskProcessor := worker.NewRedisTaskProcessor(redisOpt, store, sender, config)

	l.Info("start task processor")

	if err := taskProcessor.Start(); err != nil {
		l.Fatal("cannot start task processor", zap.Error(err))
	}
}

// runGinServer starts the Gin HTTP REST server (kept for reference).
func runGinServer(store *db.Store, config util.Config) {
	server, err := api.NewServer(store, config)
	if err != nil {
		logger.G().Fatal("cannot create Gin server", zap.Error(err))
	}

	address := fmt.Sprintf("%s:%s", config.SERVER_ADDRESS, config.PORT)
	if err := server.Start(address); err != nil {
		logger.G().Fatal("cannot start Gin server", zap.Error(err))
	}
}

// runGRPCServer starts the gRPC server with the auth + logging interceptors.
func runGRPCServer(store *db.Store, config util.Config, taskDistributor worker.TaskDistributor) {
	l := logger.G()

	server, err := gapi.NewServer(store, config, taskDistributor)
	if err != nil {
		l.Fatal("cannot create gRPC server", zap.Error(err))
	}

	grpcOpts := logger.GRPCLogOptions{
		Logger: l,
		SkipMethods: map[string]bool{
			"/grpc.health.v1.Health/Check": true,
		},
		// LogPayloads: true, // Enable only in development
	}

	grpcServer := grpc.NewServer(
		grpc.ChainUnaryInterceptor(
			logger.UnaryServerInterceptor(grpcOpts),
			server.AuthInterceptor(),
		),
		grpc.ChainStreamInterceptor(
			logger.StreamServerInterceptor(grpcOpts),
		),
	)

	pb.RegisterGoBankServer(grpcServer, server)
	reflection.Register(grpcServer)

	address := fmt.Sprintf("%s:%s", config.SERVER_ADDRESS, config.GRPC_SERVER_PORT)
	listener, err := net.Listen("tcp", address)
	if err != nil {
		l.Fatal("cannot create gRPC listener", zap.Error(err))
	}

	l.Info("gRPC server listening", zap.String("addr", listener.Addr().String()))

	if err := grpcServer.Serve(listener); err != nil {
		l.Fatal("gRPC server stopped", zap.Error(err))
	}
}

// runGatewayServer starts the gRPC-Gateway HTTP server with the HTTP logger middleware.
func runGatewayServer(config util.Config) {
	l := logger.G()

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
			lower := strings.ToLower(key)
			// Pass through auth, correlation, and trace headers.
			switch lower {
			case "authorization", "x-request-id", "x-trace-id", "x-span-id":
				return lower, true
			}
			return runtime.DefaultHeaderMatcher(key)
		}),
	)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// ← This routes HTTP → actual gRPC server (interceptor runs)
	// instead of RegisterGoBankHandlerServer which bypasses interceptor
	grpcEndpoint := fmt.Sprintf("%s:%s", config.SERVER_ADDRESS, config.GRPC_SERVER_PORT)
	err := pb.RegisterGoBankHandlerFromEndpoint(ctx, grpcMux, grpcEndpoint, []grpc.DialOption{
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	})
	if err != nil {
		l.Fatal("cannot register gateway handler", zap.Error(err))
	}

	httpOpts := logger.HTTPLogOptions{
		Logger: l,
		SkipPaths: []string{
			"/healthz",
			"/readyz",
		},
		ObservabilityPaths: []string{
			"/metrics",
		},
		SkipPathPrefixes: []string{
			"/swagger/", // already served statically, no need to log each asset
		},
		// LogRequestBody: true, // dev only
	}


	mux := http.NewServeMux()
	mux.HandleFunc("/swagger/", gapi.SwaggerHandler)
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	mux.HandleFunc("/readyz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	mux.Handle("/", grpcMux)

	loggedMux := logger.HTTPMiddleware(httpOpts)(mux)

	httpAddress := fmt.Sprintf("%s:%s", config.SERVER_ADDRESS, config.PORT)
	listener, err := net.Listen("tcp", httpAddress)
	if err != nil {
		l.Fatal("cannot create HTTP listener", zap.Error(err))
	}

	l.Info("HTTP gateway listening",
		zap.String("addr", fmt.Sprintf("http://%s", httpAddress)),
		zap.String("swagger", fmt.Sprintf("http://%s/swagger/", httpAddress)),
	)

	if err = http.Serve(listener, loggedMux); err != nil {
		l.Fatal("HTTP gateway stopped", zap.Error(err))
	}
}