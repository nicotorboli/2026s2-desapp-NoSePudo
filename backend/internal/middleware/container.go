package middleware

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"log/slog"
	"net/http"
	"time"

	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/logger"
)

type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

type Middleware func(http.Handler) http.Handler

type Container struct {
	logger *slog.Logger
}

func NewContainer(l *slog.Logger) *Container {
	return &Container{
		logger: l,
	}
}

// CorrelationID extracts or generates an X-Correlation-ID and stores it in context.
func CorrelationID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		corrID := r.Header.Get("X-Correlation-ID")
		if corrID == "" {
			var b [16]byte
			_, _ = rand.Read(b[:])
			corrID = hex.EncodeToString(b[:])
		}

		ctx := context.WithValue(r.Context(), logger.CorrelationIDContextKey, corrID)
		w.Header().Set("X-Correlation-ID", corrID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// RequestLogger logs incoming requests with status code, duration, and correlation ID.
func RequestLogger(l *slog.Logger) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			wrapped := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

			next.ServeHTTP(wrapped, r)

			duration := time.Since(start)
			l.InfoContext(r.Context(), "HTTP Request",
				"method", r.Method,
				"path", r.URL.Path,
				"status", wrapped.statusCode,
				"duration_ms", duration.Milliseconds(),
			)
		})
	}
}

// CORS provides permissive CORS headers for development.
func CORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Correlation-ID, X-Auth-Token")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// Chain applies middlewares in standard execution order.
func Chain(h http.Handler, middlewares ...Middleware) http.Handler {
	for i := len(middlewares) - 1; i >= 0; i-- {
		h = middlewares[i](h)
	}
	return h
}
