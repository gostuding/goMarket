package middlewares

import (
	"compress/gzip"
	"fmt"
	"io"
	"net/http"
	"strings"

	"go.uber.org/zap"
)

type gzipWriter struct {
	http.ResponseWriter
	logger    *zap.SugaredLogger
	isWriting bool
}

func newGzipWriter(r http.ResponseWriter, logger *zap.SugaredLogger) *gzipWriter {
	return &gzipWriter{ResponseWriter: r, isWriting: false, logger: logger}
}

func (r *gzipWriter) Write(b []byte) (int, error) {
	if !r.isWriting && r.Header().Get(ceString) == gzipString {
		r.isWriting = true
		compressor := gzip.NewWriter(r)
		size, err := compressor.Write(b)
		if err != nil {
			return 0, fmt.Errorf("compress respons body error: %w", err)
		}
		if err = compressor.Close(); err != nil {
			return 0, fmt.Errorf("compress close error: %w", err)
		}
		r.isWriting = false
		return size, nil
	}
	return r.ResponseWriter.Write(b) //nolint:wrapcheck // <- default action
}

func (r *gzipWriter) WriteHeader(statusCode int) {
	contentType := r.Header().Get(ctString) == "application/json" || r.Header().Get(ctString) == "text/html"
	if statusCode == 200 && contentType {
		r.Header().Set(ceString, gzipString)
	}
	r.ResponseWriter.WriteHeader(statusCode)
}

func (r *gzipWriter) Header() http.Header {
	return r.ResponseWriter.Header()
}

type gzipReader struct {
	r    io.ReadCloser
	gzip *gzip.Reader
}

func newGzipReader(r io.ReadCloser) (*gzipReader, error) {
	reader, err := gzip.NewReader(r)
	if err != nil {
		return nil, fmt.Errorf("create gzip reader error: %w", err)
	}
	return &gzipReader{r: r, gzip: reader}, nil
}

func (c gzipReader) Read(p []byte) (n int, err error) {
	return c.gzip.Read(p) //nolint:wrapcheck // <- default action
}

func (c *gzipReader) Close() error {
	if err := c.r.Close(); err != nil {
		return fmt.Errorf("close gzip reader error: %w", err)
	}
	return c.gzip.Close() //nolint:wrapcheck // <- default action
}

func GzipMiddleware(logger *zap.SugaredLogger) func(h http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {
			if strings.Contains(r.Header.Get(ceString), gzipString) {
				cr, err := newGzipReader(r.Body)
				if err != nil {
					w.WriteHeader(http.StatusInternalServerError)
					logger.Warnf("gzip reader create error: %w", err)
					return
				}
				r.Body = cr
				defer cr.Close() //nolint:errcheck // <- senselessly
			}
			if strings.Contains(r.Header.Get("Accept-Encoding"), gzipString) {
				next.ServeHTTP(newGzipWriter(w, logger), r)
			} else {
				next.ServeHTTP(w, r)
			}
		}
		return http.HandlerFunc(fn)
	}
}
