package middlewares

import (
	"compress/gzip"
	"fmt"
	"net/http"
	"strings"
)

func Compression(next http.Handler) http.Handler {
	fmt.Println("Compression Middleware...")
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Println("Compression Middleware being returned...")

		// check if the client contains gzip encoding
		if !strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") {

			next.ServeHTTP(w, r)
		}

		// set the response header
		w.Header().Set("Content-Encoding", "gzip")
		gz := gzip.NewWriter(w)
		defer gz.Close()

		// wrap the response writer
		w = &gzipResponseWriter{ResponseWriter: w, Writer: gz}

		next.ServeHTTP(w, r)
		fmt.Println("Response From Compression Middleware")
	})
}

// gzip response writer will wrap the http.ResponseWriter to write gzipped responses
type gzipResponseWriter struct {
	http.ResponseWriter
	Writer *gzip.Writer
}

func (g *gzipResponseWriter) Write(b []byte) (int, error) {
	return g.Writer.Write(b)
}
