package middlewares

import (
	"fmt"
	"net/http"
	"time"
)

func ResponseTimeMiddleware(next http.Handler) http.Handler {
	fmt.Println("Response Time Middleware...")
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Println("Response Time Middleware Being Returned...")
		fmt.Println("Recieved Request in Response Time")
		start := time.Now()

		// create a custome responsewriter to capture the status code
		wrappedWriter := &responseWriter{ResponseWriter: w, status: http.StatusOK}

		// calculate the duration
		duration := time.Since(start)

		w.Header().Set("X-Response-Time", duration.String())
		next.ServeHTTP(wrappedWriter, r)

		duration = time.Since(start)

		// log the request details
		fmt.Printf("Method: %s, URL: %s, Status: %d, Duration: %v\n", r.Method, r.URL, wrappedWriter.status, duration.String())
		fmt.Println("Sent response from Response Time Middleware")
	})
}

// response writer
type responseWriter struct {
	http.ResponseWriter
	status int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.status = code
	rw.ResponseWriter.WriteHeader(code)
}
