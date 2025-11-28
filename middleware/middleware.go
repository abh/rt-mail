package middleware

import (
	"fmt"
	"net/http"
	"time"

	"go.ntppool.org/common/logger"
)

// Recovery middleware recovers from panics
func Recovery(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				ctx := r.Context()
				log := logger.FromContext(ctx)
				log.ErrorContext(ctx, "panic recovered", "error", fmt.Sprintf("%v", err))
				http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			}
		}()
		next.ServeHTTP(w, r)
	})
}

// Logging middleware logs requests
func Logging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Skip logging for healthz endpoint
		if r.URL.Path == "/healthz" {
			next.ServeHTTP(w, r)
			return
		}

		start := time.Now()
		wrapped := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

		next.ServeHTTP(wrapped, r)

		duration := time.Since(start)
		ctx := r.Context()
		log := logger.FromContext(ctx)

		log.InfoContext(ctx, "http request",
			"remote_addr", r.RemoteAddr,
			"method", r.Method,
			"path", r.URL.Path,
			"proto", r.Proto,
			"status", wrapped.statusCode,
			"bytes", wrapped.bytesWritten,
			"referer", r.Referer(),
			"user_agent", r.UserAgent(),
			"duration_ms", float64(duration.Nanoseconds())/1e6,
		)
	})
}

type responseWriter struct {
	http.ResponseWriter
	statusCode   int
	bytesWritten int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

func (rw *responseWriter) Write(b []byte) (int, error) {
	n, err := rw.ResponseWriter.Write(b)
	rw.bytesWritten += n
	return n, err
}

// Unwrap returns the underlying ResponseWriter for http.ResponseController compatibility.
func (rw *responseWriter) Unwrap() http.ResponseWriter {
	return rw.ResponseWriter
}

// Chain applies middleware in order
func Chain(h http.Handler, middleware ...func(http.Handler) http.Handler) http.Handler {
	for i := len(middleware) - 1; i >= 0; i-- {
		h = middleware[i](h)
	}
	return h
}
