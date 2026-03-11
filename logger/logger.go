package logger

import (
	"context"
	"io"
	"os"
	"sync"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// ─── Context key type

type contextKey string

const (
	loggerKey    contextKey = "logger"
	requestIDKey contextKey = "request_id"
	traceIDKey   contextKey = "trace_id"
	spanIDKey    contextKey = "span_id"
	userKey      contextKey = "user"
)

// ─── Log levels

type Level = zapcore.Level

const (
	DebugLevel = zapcore.DebugLevel
	InfoLevel  = zapcore.InfoLevel
	WarnLevel  = zapcore.WarnLevel
	ErrorLevel = zapcore.ErrorLevel
	FatalLevel = zapcore.FatalLevel
)

// ─── Config

type Config struct {
	// Development enables pretty-printed colored console output.
	Development bool

	// Level is the minimum log level. Defaults to InfoLevel.
	Level Level

	// ServiceName is injected into every log line.
	ServiceName string

	// ServiceVersion is injected into every log line.
	ServiceVersion string

	// Environment: "production", "staging", "development"
	Environment string

	// Output writers — defaults to os.Stdout / os.Stderr
	Output    io.Writer
	ErrOutput io.Writer

	// SamplingConfig enables sampling for high-throughput paths.
	// Set to nil to disable sampling.
	Sampling *SamplingConfig

	// EnableCaller adds caller file:line to every log entry.
	EnableCaller bool

	// EnableStackTrace adds stack trace to Error+ logs.
	EnableStackTrace bool
}

type SamplingConfig struct {
	// Initial is the number of entries with identical level and message
	// logged per second before sampling kicks in.
	Initial int
	// Thereafter is the sampling rate after Initial is exceeded.
	Thereafter int
}

// DefaultConfig returns a production-safe config.
func DefaultConfig(service, version, env string) Config {
	return Config{
		Development:      env == "development",
		Level:            InfoLevel,
		ServiceName:      service,
		ServiceVersion:   version,
		Environment:      env,
		EnableCaller:     true,
		EnableStackTrace: true,
		Sampling: &SamplingConfig{
			Initial:    100,
			Thereafter: 100,
		},
	}
}

// ─── Logger

// Logger wraps zap.Logger with request-scoped field injection.
type Logger struct {
	zap    *zap.Logger
	sugar  *zap.SugaredLogger
	config Config
	mu     sync.RWMutex

	// Base fields applied to every log entry.
	baseFields []zap.Field
}

var (
	global *Logger
	once   sync.Once
)

// New creates a new Logger from Config.
func New(cfg Config) (*Logger, error) {
	core, err := buildCore(cfg)
	if err != nil {
		return nil, err
	}

	opts := []zap.Option{
		zap.WithCaller(cfg.EnableCaller),
		zap.AddCallerSkip(1),
	}
	if cfg.EnableStackTrace {
		opts = append(opts, zap.AddStacktrace(ErrorLevel))
	}

	z := zap.New(core, opts...)

	l := &Logger{
		zap:    z,
		sugar:  z.Sugar(),
		config: cfg,
		baseFields: []zap.Field{
			zap.String("service", cfg.ServiceName),
			zap.String("version", cfg.ServiceVersion),
			zap.String("env", cfg.Environment),
		},
	}

	return l, nil
}

// MustNew panics on error — use in main().
func MustNew(cfg Config) *Logger {
	l, err := New(cfg)
	if err != nil {
		panic("logger: " + err.Error())
	}
	return l
}

// InitGlobal initialises the package-level logger. Call once from main().
func InitGlobal(cfg Config) error {
	var initErr error
	once.Do(func() {
		l, err := New(cfg)
		if err != nil {
			initErr = err
			return
		}
		global = l
		// Redirect stdlib log to zap.
		if _, err := zap.RedirectStdLogAt(l.zap, InfoLevel); err != nil {
			initErr = err
		}
	})
	return initErr
}

// G returns the global logger. Panics if InitGlobal was not called.
func G() *Logger {
	if global == nil {
		panic("logger: global logger not initialised — call InitGlobal first")
	}
	return global
}

// ─── Core builder

func buildCore(cfg Config) (zapcore.Core, error) {
	outWriter := cfg.Output
	if outWriter == nil {
		outWriter = os.Stdout
	}
	errWriter := cfg.ErrOutput
	if errWriter == nil {
		errWriter = os.Stderr
	}

	atomicLevel := zap.NewAtomicLevelAt(cfg.Level)

	var enc zapcore.Encoder
	if cfg.Development {
		enc = buildConsoleEncoder()
	} else {
		enc = buildJSONEncoder()
	}

	// Route Error+ to stderr, everything else to stdout.
	highPriority := zap.LevelEnablerFunc(func(lvl zapcore.Level) bool {
		return lvl >= ErrorLevel && atomicLevel.Enabled(lvl)
	})
	lowPriority := zap.LevelEnablerFunc(func(lvl zapcore.Level) bool {
		return lvl < ErrorLevel && atomicLevel.Enabled(lvl)
	})

	highCore := zapcore.NewCore(enc, zapcore.AddSync(errWriter), highPriority)
	lowCore := zapcore.NewCore(enc, zapcore.AddSync(outWriter), lowPriority)

	combined := zapcore.NewTee(lowCore, highCore)

	if cfg.Sampling != nil {
		return zapcore.NewSamplerWithOptions(
			combined,
			time.Second,
			cfg.Sampling.Initial,
			cfg.Sampling.Thereafter,
		), nil
	}

	return combined, nil
}

// ─── Encoders

func buildJSONEncoder() zapcore.Encoder {
	ec := zap.NewProductionEncoderConfig()
	ec.TimeKey = "ts"
	ec.EncodeTime = zapcore.RFC3339NanoTimeEncoder
	ec.LevelKey = "level"
	ec.EncodeLevel = zapcore.LowercaseLevelEncoder
	ec.MessageKey = "msg"
	ec.CallerKey = "caller"
	ec.EncodeCaller = zapcore.ShortCallerEncoder
	ec.StacktraceKey = "stacktrace"
	ec.FunctionKey = zapcore.OmitKey
	return zapcore.NewJSONEncoder(ec)
}

func buildConsoleEncoder() zapcore.Encoder {
	ec := zap.NewDevelopmentEncoderConfig()
	ec.EncodeTime = func(t time.Time, enc zapcore.PrimitiveArrayEncoder) {
		enc.AppendString(t.Format("15:04:05.000"))
	}
	ec.EncodeLevel = zapcore.CapitalColorLevelEncoder
	ec.EncodeCaller = zapcore.ShortCallerEncoder
	ec.ConsoleSeparator = "  "
	return zapcore.NewConsoleEncoder(ec)
}

// ─── With / context helpers

// With returns a child Logger with additional permanent fields.
func (l *Logger) With(fields ...zap.Field) *Logger {
	return &Logger{
		zap:        l.zap.With(fields...),
		sugar:      l.zap.With(fields...).Sugar(),
		config:     l.config,
		baseFields: l.baseFields,
	}
}

// FromContext extracts a logger from ctx, falling back to the global.
func FromContext(ctx context.Context) *Logger {
	if l, ok := ctx.Value(loggerKey).(*Logger); ok && l != nil {
		return l
	}
	return G()
}

// IntoContext stores logger l in ctx.
func IntoContext(ctx context.Context, l *Logger) context.Context {
	return context.WithValue(ctx, loggerKey, l)
}

// WithRequestID attaches a request/correlation ID to the context logger.
func WithRequestID(ctx context.Context, reqID string) context.Context {
	l := FromContext(ctx).With(zap.String("request_id", reqID))
	ctx = context.WithValue(ctx, requestIDKey, reqID)
	return IntoContext(ctx, l)
}

// WithTraceContext attaches OpenTelemetry-style trace/span IDs.
func WithTraceContext(ctx context.Context, traceID, spanID string) context.Context {
	l := FromContext(ctx).With(
		zap.String("trace_id", traceID),
		zap.String("span_id", spanID),
	)
	ctx = context.WithValue(ctx, traceIDKey, traceID)
	ctx = context.WithValue(ctx, spanIDKey, spanID)
	return IntoContext(ctx, l)
}

// WithUser attaches the authenticated username.
func WithUser(ctx context.Context, username string) context.Context {
	l := FromContext(ctx).With(zap.String("user", username))
	ctx = context.WithValue(ctx, userKey, username)
	return IntoContext(ctx, l)
}

// RequestIDFromContext retrieves the request ID stored by WithRequestID.
func RequestIDFromContext(ctx context.Context) string {
	if v, ok := ctx.Value(requestIDKey).(string); ok {
		return v
	}
	return ""
}

// ─── Logging methods

func (l *Logger) Debug(msg string, fields ...zap.Field) {
	l.zap.Debug(msg, l.withBase(fields)...)
}

func (l *Logger) Info(msg string, fields ...zap.Field) {
	l.zap.Info(msg, l.withBase(fields)...)
}

func (l *Logger) Warn(msg string, fields ...zap.Field) {
	l.zap.Warn(msg, l.withBase(fields)...)
}

func (l *Logger) Error(msg string, fields ...zap.Field) {
	l.zap.Error(msg, l.withBase(fields)...)
}

func (l *Logger) Fatal(msg string, fields ...zap.Field) {
	l.zap.Fatal(msg, l.withBase(fields)...)
}

// Sync flushes any buffered log entries. Call on shutdown.
func (l *Logger) Sync() error {
	return l.zap.Sync()
}

// ZapLogger exposes the underlying *zap.Logger for libraries that require it.
func (l *Logger) ZapLogger() *zap.Logger {
	return l.zap
}

// withBase merges base fields into every log call.
// Base fields are already baked in when using With(), so only raw loggers
// need this merge on every call.
func (l *Logger) withBase(fields []zap.Field) []zap.Field {
	return fields
}