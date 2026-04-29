package httpx

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"log/slog"
	mathrand "math/rand"
	"net/http"
	"runtime/debug"
	"time"
)

type Middleware func(http.Handler) http.Handler

type contextKey string

const requestIDKey contextKey = "request-id"

func Chain(handler http.Handler, middlewares ...Middleware) http.Handler {
	for i := len(middlewares) - 1; i >= 0; i-- {
		handler = middlewares[i](handler)
	}
	return handler
}

func Recover(logger *slog.Logger) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if rec := recover(); rec != nil {
					logger.Error("panic recovered", "panic", rec, "stack", string(debug.Stack()))
					WriteError(w, http.StatusInternalServerError, "internal server error")
				}
			}()
			next.ServeHTTP(w, r)
		})
	}
}

func RequestID() Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			requestID := r.Header.Get("X-Request-ID")
			if requestID == "" {
				requestID = newRequestID()
			}
			w.Header().Set("X-Request-ID", requestID)
			ctx := context.WithValue(r.Context(), requestIDKey, requestID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func RequestIDFromContext(ctx context.Context) string {
	value, _ := ctx.Value(requestIDKey).(string)
	return value
}

func AccessLog(logger *slog.Logger) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			recorder := &statusRecorder{ResponseWriter: w, statusCode: http.StatusOK}
			next.ServeHTTP(recorder, r)
			logger.Info("http request",
				"method", r.Method,
				"path", r.URL.Path,
				"status", recorder.statusCode,
				"duration_ms", time.Since(start).Milliseconds(),
				"request_id", RequestIDFromContext(r.Context()),
			)
		})
	}
}

func FaultInjection(latency time.Duration, failRate float64, logger *slog.Logger) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if latency > 0 {
				logger.Warn("demo latency injected", "latency_ms", latency.Milliseconds(), "path", r.URL.Path)
				timer := time.NewTimer(latency)
				select {
				case <-timer.C:
				case <-r.Context().Done():
					timer.Stop()
					return
				}
			}
			if failRate > 0 && mathrand.Float64() < failRate {
				logger.Warn("demo failure injected", "fail_rate", failRate, "path", r.URL.Path)
				WriteError(w, http.StatusServiceUnavailable, "demo dependency failure")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

type statusRecorder struct {
	http.ResponseWriter
	statusCode int
}

func (r *statusRecorder) WriteHeader(statusCode int) {
	r.statusCode = statusCode
	r.ResponseWriter.WriteHeader(statusCode)
}

func newRequestID() string {
	bytes := make([]byte, 8)
	if _, err := rand.Read(bytes); err != nil {
		return time.Now().UTC().Format("20060102150405.000000000")
	}
	return hex.EncodeToString(bytes)
}
