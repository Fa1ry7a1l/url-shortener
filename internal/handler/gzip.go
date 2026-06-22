package handler

import (
	"compress/gzip"
	"io"
	"net/http"
	"strings"
)

type gzipResponseWriter struct {
	http.ResponseWriter
	writer        io.Writer
	gzipWriter    *gzip.Writer
	statusCode    int
	headerWritten bool
}

func (w *gzipResponseWriter) WriteHeader(statusCode int) {
	if w.headerWritten {
		return
	}
	w.statusCode = statusCode
	w.headerWritten = true

	contentType := w.Header().Get("Content-Type")
	if shouldCompressContentType(contentType) {
		w.Header().Set("Content-Encoding", "gzip")
		w.Header().Del("Content-Length")
		w.gzipWriter = gzip.NewWriter(w.ResponseWriter)
		w.writer = w.gzipWriter
	} else {
		w.writer = w.ResponseWriter
	}

	w.ResponseWriter.WriteHeader(statusCode)
}

func (w *gzipResponseWriter) Write(p []byte) (int, error) {
	if !w.headerWritten {
		w.WriteHeader(http.StatusOK)
	}
	return w.writer.Write(p)
}

func (w *gzipResponseWriter) Close() error {
	if w.gzipWriter != nil {
		return w.gzipWriter.Close()
	}
	return nil
}

func shouldCompressContentType(contentType string) bool {
	contentType = strings.ToLower(contentType)

	return strings.HasPrefix(contentType, "application/json") ||
		strings.HasPrefix(contentType, "text/html")
}

func acceptsGzip(r *http.Request) bool {
	return strings.Contains(strings.ToLower(r.Header.Get("Accept-Encoding")), "gzip")
}

func isGzipEncoded(r *http.Request) bool {
	return strings.Contains(strings.ToLower(r.Header.Get("Content-Encoding")), "gzip")
}

// GzipHandle returns middleware that decompresses gzip requests and compresses supported responses.
func GzipHandle(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Распаковка входящего запроса
		if isGzipEncoded(r) {
			gz, err := gzip.NewReader(r.Body)
			if err != nil {
				http.Error(w, "bad request", http.StatusBadRequest)
				return
			}
			defer gz.Close()

			r.Body = &readCloserWrapper{
				Reader: gz,
				Closer: r.Body,
			}
		}

		// Если клиент не просил gzip-ответ — просто идём дальше
		if !acceptsGzip(r) {
			next.ServeHTTP(w, r)
			return
		}

		gzw := &gzipResponseWriter{
			ResponseWriter: w,
			writer:         w,
		}
		defer func() {
			_ = gzw.Close()
		}()

		next.ServeHTTP(gzw, r)
	})
}

type readCloserWrapper struct {
	io.Reader
	Closer io.Closer
}

func (r *readCloserWrapper) Close() error {
	return r.Closer.Close()
}
