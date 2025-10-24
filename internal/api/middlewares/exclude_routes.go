package middlewares

import (
	"fmt"
	"net/http"
	"strings"
)

func MiddlewaresExcludePaths(middleware func(http.Handler) http.Handler, excludedPaths ...string) func(http.Handler) http.Handler {
	fmt.Println("MiddlewaresExcludePaths initialised")
	return func(next http.Handler) http.Handler {
		fmt.Println("=========== MiddlewaresExcludePaths RAN")
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			for _, path := range excludedPaths {
				if strings.HasPrefix(r.URL.Path, path) {
					next.ServeHTTP(w, r)
					return
				}
			}
			middleware(next).ServeHTTP(w, r)
			fmt.Println("=========== sending response from MiddlewaresExcludePaths")
		})
	}
}
