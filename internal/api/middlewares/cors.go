package middlewares

import (
	"fmt"
	"net/http"
	"qiyana_paybuddy/pkg/utils"
)

// Allowed origins
var allowedOrigins = []string{
	"https://my-origin-url.com",
	"https://localhost:3000",
}

func Cors(next http.Handler) http.Handler {
	fmt.Println("Cors Middleware...")
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Println("Cors Middleware Returned...")
		origin := r.Header.Get("Origin")
		fmt.Println(origin)

		if isOriginAllowed(origin) {
			w.Header().Set("Access-Control-Allow-Origin", origin)
		} else {
			utils.WriteError(w, "Not Allowed By CORS", http.StatusForbidden)
			return
		}
		// w.Header().Set()
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		w.Header().Set("Access-Control-Expose-Headers", "Authorization")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE")
		w.Header().Set("Access-Control-Allow-Credentials", "true")
		w.Header().Set("Access-Control-Max-Age", "3600")

		if r.Method == http.MethodOptions {
			return
		}

		next.ServeHTTP(w, r)
		fmt.Println("Cors Middleware Ends...")
	})
}

func isOriginAllowed(origin string) bool {
	for _, allowedOrigin := range allowedOrigins {

		if origin == allowedOrigin {
			return true
		}
	}
	return false
}
