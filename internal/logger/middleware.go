package logger

import (
	"net/http"
	"time"

	"go.uber.org/zap"
)

type responseData struct {
	status int
	size   int
}

type loggingResponseWriter struct {
	http.ResponseWriter
	data *responseData
}

func (lw *loggingResponseWriter) WriteHeader(statusCode int) {
	lw.data.status = statusCode
	lw.ResponseWriter.WriteHeader(statusCode)
}

func (lw *loggingResponseWriter) Write(b []byte) (int, error) {
	if lw.data.status == 0 {
		lw.data.status = http.StatusOK
	}

	n, err := lw.ResponseWriter.Write(b)
	lw.data.size += n
	return n, err
}

// RequestLogger returns middleware that logs request method, URI, status, size, and duration.
func RequestLogger(log Logger) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			data := &responseData{}
			lw := &loggingResponseWriter{
				ResponseWriter: w,
				data:           data,
			}

			next.ServeHTTP(lw, r)

			if data.status == 0 {
				data.status = http.StatusOK
			}

			log.Info("request completed",
				zap.String("uri", r.RequestURI),
				zap.String("method", r.Method),
				zap.Duration("duration", time.Since(start)),
				zap.Int("status", data.status),
				zap.Int("size", data.size),
			)
		})
	}
}
