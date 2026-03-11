package logger

import (
	"context"
	"path"
	"strings"
	"time"

	"github.com/a7medalyapany/GoBank.git/token"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"
)

// ─── gRPC call context

type grpcFields struct {
	service   string
	method    string
	peerAddr  string
	userAgent string
	requestID string
	traceID   string
	spanID    string
	username  string
	deadline  *time.Time
}

// ─── Options

type GRPCLogOptions struct {
	// Logger to use — defaults to G() if nil.
	Logger *Logger

	// SkipMethods is a set of full gRPC method names to suppress.
	// e.g. "/grpc.health.v1.Health/Check"
	SkipMethods map[string]bool

	// LogPayloads enables request/response payload logging.
	// WARNING: may log sensitive data. Development only.
	LogPayloads bool

	// DeciderFunc overrides the default level-per-RPC logic.
	// Return log=false to suppress a call entirely.
	DeciderFunc func(fullMethod string, err error) (zapcore.Level, bool)

	// Header names for correlation/trace IDs forwarded by gRPC-Gateway.
	// Defaults: "x-request-id", "x-trace-id", "x-span-id"
	RequestIDHeader string
	TraceIDHeader   string
	SpanIDHeader    string
}

func (o *GRPCLogOptions) logger() *Logger {
	if o.Logger != nil {
		return o.Logger
	}
	return G()
}

func (o *GRPCLogOptions) requestIDHeader() string {
	if o.RequestIDHeader != "" {
		return strings.ToLower(o.RequestIDHeader)
	}
	return "x-request-id"
}

func (o *GRPCLogOptions) traceIDHeader() string {
	if o.TraceIDHeader != "" {
		return strings.ToLower(o.TraceIDHeader)
	}
	return "x-trace-id"
}

func (o *GRPCLogOptions) spanIDHeader() string {
	if o.SpanIDHeader != "" {
		return strings.ToLower(o.SpanIDHeader)
	}
	return "x-span-id"
}

func defaultDecider(_ string, err error) (zapcore.Level, bool) {
	code := status.Code(err)
	switch {
	case code == codes.OK:
		return InfoLevel, true
	case isClientError(code):
		return WarnLevel, true
	default:
		return ErrorLevel, true
	}
}

// ─── Unary interceptor

// UnaryServerInterceptor logs every unary RPC call.
//
// Single log line per call — all fields on the finish event:
//
//	{
//	  "ts":          "09:04:06.071",
//	  "level":       "warn",
//	  "msg":         "gRPC",
//	  "service":     "go-bank",
//	  "grpc.svc":    "pb.GoBank",
//	  "grpc.method": "ListAccounts",
//	  "grpc.code":   "Unauthenticated",
//	  "grpc.dur_ms": 0.21,
//	  "grpc.peer":   "127.0.0.1:6057",
//	  "request_id":  "9e40817de4f65dc2",   ← correlated with HTTP log line
//	  "user":        "alice"               ← present after auth succeeds
//	}
func UnaryServerInterceptor(opts GRPCLogOptions) grpc.UnaryServerInterceptor {
	decider := opts.DeciderFunc
	if decider == nil {
		decider = defaultDecider
	}

	return func(
		ctx context.Context,
		req any,
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (any, error) {

		if opts.SkipMethods[info.FullMethod] {
			return handler(ctx, req)
		}

		f := extractGRPCFields(ctx, info.FullMethod, opts)

		// Enrich the context logger so downstream handlers can use it.
		ctx = enrichContext(ctx, f, opts.logger())

		start := time.Now()
		resp, err := handler(ctx, req)
		elapsed := time.Since(start)

		// After handler runs, try to pick up username that authInterceptor
		// injected into ctx (it runs inside handler when chained after us).
		if f.username == "" {
			if payload, ok := ctx.Value(authPayloadContextKey).(*token.Payload); ok && payload != nil {
				f.username = payload.Username
			}
		}

		code := status.Code(err)
		level, shouldLog := decider(info.FullMethod, err)
		if !shouldLog {
			return resp, err
		}

		fields := buildFinishFields(f, code, elapsed, err)
		if opts.LogPayloads && req != nil {
			fields = append(fields, zap.Any("grpc.req", req))
		}
		if opts.LogPayloads && resp != nil {
			fields = append(fields, zap.Any("grpc.resp", resp))
		}

		logAtLevel(FromContext(ctx), level, "gRPC", fields...)
		return resp, err
	}
}

// ─── Stream interceptor

// StreamServerInterceptor logs stream open/close with message counts.
func StreamServerInterceptor(opts GRPCLogOptions) grpc.StreamServerInterceptor {
	decider := opts.DeciderFunc
	if decider == nil {
		decider = defaultDecider
	}

	return func(
		srv any,
		ss grpc.ServerStream,
		info *grpc.StreamServerInfo,
		handler grpc.StreamHandler,
	) error {

		if opts.SkipMethods[info.FullMethod] {
			return handler(srv, ss)
		}

		ctx := ss.Context()
		f := extractGRPCFields(ctx, info.FullMethod, opts)
		ctx = enrichContext(ctx, f, opts.logger())

		wrapped := &wrappedStream{ServerStream: ss, ctx: ctx}

		start := time.Now()
		err := handler(srv, wrapped)
		elapsed := time.Since(start)

		code := status.Code(err)
		level, shouldLog := decider(info.FullMethod, err)
		if !shouldLog {
			return err
		}

		fields := append(
			buildFinishFields(f, code, elapsed, err),
			zap.Int64("grpc.msgs_sent", wrapped.msgsSent),
			zap.Int64("grpc.msgs_recv", wrapped.msgsRecv),
			zap.Bool("grpc.client_stream", info.IsClientStream),
			zap.Bool("grpc.server_stream", info.IsServerStream),
		)
		logAtLevel(FromContext(ctx), level, "gRPC stream", fields...)
		return err
	}
}

// wrappedStream counts messages on a bidirectional stream.
type wrappedStream struct {
	grpc.ServerStream
	ctx      context.Context
	msgsSent int64
	msgsRecv int64
}

func (w *wrappedStream) Context() context.Context { return w.ctx }

func (w *wrappedStream) SendMsg(m any) error {
	err := w.ServerStream.SendMsg(m)
	if err == nil {
		w.msgsSent++
	}
	return err
}

func (w *wrappedStream) RecvMsg(m any) error {
	err := w.ServerStream.RecvMsg(m)
	if err == nil {
		w.msgsRecv++
	}
	return err
}

// ─── Field builders

func extractGRPCFields(ctx context.Context, fullMethod string, opts GRPCLogOptions) grpcFields {
	svc := path.Dir(fullMethod)
	if len(svc) > 0 && svc[0] == '/' {
		svc = svc[1:]
	}

	f := grpcFields{
		service: svc,
		method:  path.Base(fullMethod),
	}

	if dl, ok := ctx.Deadline(); ok {
		f.deadline = &dl
	}

	if p, ok := peer.FromContext(ctx); ok {
		f.peerAddr = p.Addr.String()
	}

	if md, ok := metadata.FromIncomingContext(ctx); ok {
		f.requestID = firstMD(md, opts.requestIDHeader())
		f.traceID = firstMD(md, opts.traceIDHeader())
		f.spanID = firstMD(md, opts.spanIDHeader())

		// grpc-gateway forwards the browser UA under this key
		if ua := firstMD(md, "grpcgateway-user-agent"); ua != "" {
			f.userAgent = ua
		} else {
			f.userAgent = firstMD(md, "user-agent")
		}
	}

	// Username — may already be in ctx if auth ran before us in the chain.
	if payload, ok := ctx.Value(authPayloadContextKey).(*token.Payload); ok && payload != nil {
		f.username = payload.Username
	}

	return f
}

// enrichContext pushes a scoped logger (with correlation fields) into ctx.
func enrichContext(ctx context.Context, f grpcFields, base *Logger) context.Context {
	fields := []zap.Field{
		zap.String("grpc.svc", f.service),
		zap.String("grpc.method", f.method),
	}
	if f.requestID != "" {
		fields = append(fields, zap.String("request_id", f.requestID))
		ctx = context.WithValue(ctx, requestIDKey, f.requestID)
	}
	if f.traceID != "" {
		fields = append(fields, zap.String("trace_id", f.traceID))
	}
	if f.username != "" {
		fields = append(fields, zap.String("user", f.username))
	}
	return IntoContext(ctx, base.With(fields...))
}

// buildFinishFields assembles the final log fields.
// grpc.svc and grpc.method are NOT included here — they're already baked
// into the scoped logger via enrichContext → With(), so adding them again
// would produce duplicate keys.
func buildFinishFields(f grpcFields, code codes.Code, elapsed time.Duration, err error) []zap.Field {
	fields := []zap.Field{
		zap.String("grpc.code", code.String()),
		zap.Float64("grpc.dur_ms", durMS(elapsed)),
		zap.String("grpc.peer", f.peerAddr),
	}
	if f.userAgent != "" {
		fields = append(fields, zap.String("grpc.ua", f.userAgent))
	}
	if err != nil {
		fields = append(fields, zap.String("grpc.error", status.Convert(err).Message()))
	}
	if f.deadline != nil {
		remaining := time.Until(*f.deadline)
		fields = append(fields,
			zap.Duration("grpc.deadline_remaining", remaining),
			zap.Bool("grpc.deadline_exceeded", remaining < 0),
		)
	}
	return fields
}

// ─── Helpers

// authPayloadContextKey mirrors gapi.authPayloadKey without importing gapi.
type authPayloadContextKeyType string

const authPayloadContextKey authPayloadContextKeyType = "authorization_payload"

func firstMD(md metadata.MD, key string) string {
	if vals := md.Get(key); len(vals) > 0 {
		return vals[0]
	}
	return ""
}

func durMS(d time.Duration) float64 {
	return float64(d) / float64(time.Millisecond)
}

func isClientError(code codes.Code) bool {
	switch code {
	case codes.InvalidArgument,
		codes.NotFound,
		codes.AlreadyExists,
		codes.PermissionDenied,
		codes.FailedPrecondition,
		codes.Unauthenticated,
		codes.ResourceExhausted,
		codes.Canceled:
		return true
	}
	return false
}

func logAtLevel(l *Logger, level zapcore.Level, msg string, fields ...zap.Field) {
	switch level {
	case DebugLevel:
		l.Debug(msg, fields...)
	case WarnLevel:
		l.Warn(msg, fields...)
	case ErrorLevel:
		l.Error(msg, fields...)
	default:
		l.Info(msg, fields...)
	}
}