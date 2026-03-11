package logger

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// ─── HTTP options

// HTTPLogOptions configures the HTTP middleware behaviour.
type HTTPLogOptions struct {
	Logger             *Logger
	SkipPaths          []string
	SkipPathPrefixes   []string
	DeciderFunc        func(status int, latency time.Duration, r *http.Request) (zapcore.Level, bool)
	RequestIDHeader    string
	TraceIDHeader      string
	LogRequestBody     bool
	MaxBodyLogSize     int64
	SensitiveHeaders   []string
	ObservabilityPaths []string
}

func (o *HTTPLogOptions) logger() *Logger {
	if o.Logger != nil {
		return o.Logger
	}
	return G()
}

func (o *HTTPLogOptions) requestIDHeader() string {
	if o.RequestIDHeader != "" {
		return o.RequestIDHeader
	}
	return "X-Request-Id"
}

func (o *HTTPLogOptions) traceIDHeader() string {
	if o.TraceIDHeader != "" {
		return o.TraceIDHeader
	}
	return "X-Trace-Id"
}

func (o *HTTPLogOptions) maxBodySize() int64 {
	if o.MaxBodyLogSize > 0 {
		return o.MaxBodyLogSize
	}
	return 4096
}

func (o *HTTPLogOptions) sensitiveHeaders() map[string]bool {
	m := map[string]bool{
		"authorization": true,
		"cookie":        true,
		"set-cookie":    true,
	}
	for _, h := range o.SensitiveHeaders {
		m[strings.ToLower(h)] = true
	}
	return m
}

// ─── responseRecorder

type responseRecorder struct {
	http.ResponseWriter
	statusCode   int
	bytesWritten int64
	wroteHeader  bool
}

func newResponseRecorder(w http.ResponseWriter) *responseRecorder {
	return &responseRecorder{ResponseWriter: w, statusCode: http.StatusOK}
}

func (r *responseRecorder) WriteHeader(code int) {
	if !r.wroteHeader {
		r.statusCode = code
		r.wroteHeader = true
		r.ResponseWriter.WriteHeader(code)
	}
}

func (r *responseRecorder) Write(b []byte) (int, error) {
	if !r.wroteHeader {
		r.WriteHeader(http.StatusOK)
	}
	n, err := r.ResponseWriter.Write(b)
	r.bytesWritten += int64(n)
	return n, err
}

// ─── HTTPMiddleware

// HTTPMiddleware returns an http.Handler middleware that logs every request.
//
// Log fields (JSON prod / pretty dev):
//
//	ts, level, msg="HTTP finished"
//	service, env, version          ← from base logger
//	request_id, trace_id           ← from headers or generated
//	http.method, http.path         ← request routing
//	http.status, http.dur_ms       ← response
//	http.bytes, http.req_bytes     ← payload sizes
//	http.ip, http.proto, http.ua   ← client identity
//	http.referer                   ← optional
//	http.headers                   ← dev only, sensitive fields redacted
func HTTPMiddleware(opts HTTPLogOptions) func(http.Handler) http.Handler {
	skipPaths := make(map[string]bool, len(opts.SkipPaths))
	for _, p := range opts.SkipPaths {
		skipPaths[p] = true
	}

	obsPaths := make(map[string]bool, len(opts.ObservabilityPaths))
	for _, p := range opts.ObservabilityPaths {
		obsPaths[p] = true
	}

	sensitive := opts.sensitiveHeaders()

	decider := opts.DeciderFunc
	if decider == nil {
		decider = defaultHTTPDecider
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			cleanPath := r.URL.Path

			// ── Skip list
			if skipPaths[cleanPath] {
				next.ServeHTTP(w, r)
				return
			}
			for _, prefix := range opts.SkipPathPrefixes {
				if strings.HasPrefix(cleanPath, prefix) {
					next.ServeHTTP(w, r)
					return
				}
			}

			// ── Correlation IDs
			reqID := r.Header.Get(opts.requestIDHeader())
			if reqID == "" {
				reqID = generateRequestID()
			}
			w.Header().Set(opts.requestIDHeader(), reqID)
			traceID := r.Header.Get(opts.traceIDHeader())

			// ── Request-scoped logger
			ctx := r.Context()
			ctx = context.WithValue(ctx, requestIDKey, reqID)
			if traceID != "" {
				ctx = context.WithValue(ctx, traceIDKey, traceID)
			}

			baseFields := []zap.Field{
				zap.String("request_id", reqID),
				zap.String("http.method", r.Method),
				zap.String("http.path", sanitisePath(r.URL)),
			}
			if traceID != "" {
				baseFields = append(baseFields, zap.String("trace_id", traceID))
			}

			scopedLogger := opts.logger().With(baseFields...)
			ctx = IntoContext(ctx, scopedLogger)
			r = r.WithContext(ctx)

			// ── Arrival log
			scopedLogger.Debug("HTTP started",
				zap.String("http.proto", r.Proto),
				zap.String("http.ip", realIP(r)),
				zap.String("http.host", r.Host),
			)

			// ── Body capture (dev only)
			if opts.LogRequestBody && r.Body != nil {
				var buf bytes.Buffer
				limited := http.MaxBytesReader(w, r.Body, opts.maxBodySize())
				if _, err := buf.ReadFrom(limited); err == nil {
					scopedLogger.Debug("HTTP request body",
						zap.ByteString("http.body", buf.Bytes()),
					)
				}
				r.Body = io.NopCloser(&buf)
			}

			// ── Call handler
			rec := newResponseRecorder(w)
			next.ServeHTTP(rec, r)
			elapsed := time.Since(start)

			// ── Level decision
			level, shouldLog := decider(rec.statusCode, elapsed, r)
			if obsPaths[cleanPath] && level == InfoLevel {
				level = DebugLevel
			}
			if !shouldLog {
				return
			}

			// ── Finish fields
			finishFields := []zap.Field{
				zap.Int("http.status", rec.statusCode),
				zap.Float64("http.dur_ms", durMS(elapsed)),
				zap.Int64("http.bytes", rec.bytesWritten),
				zap.String("http.ip", realIP(r)),
				zap.String("http.proto", r.Proto),
			}
			if ua := r.UserAgent(); ua != "" {
				finishFields = append(finishFields, zap.String("http.ua", ua))
			}
			if ref := r.Referer(); ref != "" {
				finishFields = append(finishFields, zap.String("http.referer", ref))
			}
			if r.ContentLength > 0 {
				finishFields = append(finishFields, zap.Int64("http.req_bytes", r.ContentLength))
			}
			if opts.logger().config.Development {
				headers := make(map[string]string, len(r.Header))
				for k, v := range r.Header {
					if sensitive[strings.ToLower(k)] {
						headers[k] = "[REDACTED]"
					} else {
						headers[k] = strings.Join(v, ", ")
					}
				}
				finishFields = append(finishFields, zap.Any("http.headers", headers))
			}

			logAtLevel(scopedLogger, level, "HTTP finished", finishFields...)
		})
	}
}

// ─── Helpers

func defaultHTTPDecider(status int, _ time.Duration, _ *http.Request) (zapcore.Level, bool) {
	switch {
	case status >= 500:
		return ErrorLevel, true
	case status >= 400:
		return WarnLevel, true
	default:
		return InfoLevel, true
	}
}

// realIP extracts the actual client IP respecting common proxy headers:
// CF-Connecting-IP → X-Real-Ip → first item of X-Forwarded-For → RemoteAddr
func realIP(r *http.Request) string {
	if ip := r.Header.Get("CF-Connecting-IP"); ip != "" {
		return ip
	}
	if ip := r.Header.Get("X-Real-Ip"); ip != "" {
		return ip
	}
	if fwd := r.Header.Get("X-Forwarded-For"); fwd != "" {
		if idx := strings.IndexByte(fwd, ','); idx >= 0 {
			return strings.TrimSpace(fwd[:idx])
		}
		return strings.TrimSpace(fwd)
	}
	addr := r.RemoteAddr
	if idx := strings.LastIndexByte(addr, ':'); idx >= 0 {
		return addr[:idx]
	}
	return addr
}

// sanitisePath logs only the path, never the query string (which may carry tokens).
func sanitisePath(u *url.URL) string {
	return u.Path
}

// generateRequestID produces a crypto-random 32-char hex string.
func generateRequestID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}