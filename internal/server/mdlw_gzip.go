package server

import (
	"compress/gzip"
	"io"
	"net/http"
	"strings"

	"go.uber.org/zap"
)

type gzipWriter struct {
	http.ResponseWriter
	isWriting bool
	logger    *zap.SugaredLogger
}

func newGzipWriter(r http.ResponseWriter, logger *zap.SugaredLogger) *gzipWriter {
	return &gzipWriter{ResponseWriter: r, isWriting: false, logger: logger}
}

func (r *gzipWriter) Write(b []byte) (int, error) {
	if !r.isWriting && r.Header().Get("Content-Encoding") == "gzip" {
		r.isWriting = true
		compressor := gzip.NewWriter(r)
		size, err := compressor.Write(b)
		if err != nil {
			r.logger.Warnf("compress respons body error: %w \n", err)
			return 0, err
		}
		if err = compressor.Close(); err != nil {
			r.logger.Warnf("compress close error: %w \n", err)
			return 0, err
		}
		r.isWriting = false
		return size, err
	}
	return r.ResponseWriter.Write(b)
}

func (r *gzipWriter) WriteHeader(statusCode int) {
	contentType := r.Header().Get("Content-Type") == "application/json" || r.Header().Get("Content-Type") == "text/html"
	if statusCode == 200 && contentType {
		r.Header().Set("Content-Encoding", "gzip")
	}
	r.ResponseWriter.WriteHeader(statusCode)
}

func (r *gzipWriter) Header() http.Header {
	return r.ResponseWriter.Header()
}

// ----------------------------------------------------------------------
type gzipReader struct {
	r    io.ReadCloser
	gzip *gzip.Reader
}

func newGzipReader(r io.ReadCloser) (*gzipReader, error) {
	reader, err := gzip.NewReader(r)
	if err != nil {
		return nil, err
	}
	return &gzipReader{r: r, gzip: reader}, nil
}

func (c gzipReader) Read(p []byte) (n int, err error) {
	return c.gzip.Read(p)
}

func (c *gzipReader) Close() error {
	if err := c.r.Close(); err != nil {
		return err
	}
	return c.gzip.Close()
}

// ----------------------------------------------------------------------
func gzipMiddleware(logger *zap.SugaredLogger) func(h http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {
			if strings.Contains(r.Header.Get("Content-Encoding"), "gzip") {
				cr, err := newGzipReader(r.Body)
				if err != nil {
					w.WriteHeader(http.StatusInternalServerError)
					logger.Warnf("gzip reader create error: %w", err)
					return
				}
				r.Body = cr
				defer cr.Close()
			}
			if strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") {
				next.ServeHTTP(newGzipWriter(w, logger), r)
			} else {
				next.ServeHTTP(w, r)
			}
		}
		return http.HandlerFunc(fn)
	}
}
