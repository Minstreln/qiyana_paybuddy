package main

import (
	"crypto/tls"
	"fmt"
	"log"
	"net/http"
	"os"
	mw "qiyana_paybuddy/internal/api/middlewares"
	"qiyana_paybuddy/internal/api/routers"
	"qiyana_paybuddy/internal/repositories/sqlconnect"
	"qiyana_paybuddy/pkg/utils"

	"github.com/joho/godotenv"
)

func main() {
	err := godotenv.Load()
	if err != nil {
		return
	}

	err = sqlconnect.ConnectDb()
	if err != nil {
		utils.Logger.Fatal("DB connection failed: ", err)
	}

	utils.InitLogger()

	port := os.Getenv("SERVER_PORT")

	cert := os.Getenv("CERT_FILE")
	key := os.Getenv("KEY_FILE")

	tlsConfig := &tls.Config{
		MinVersion: tls.VersionTLS10,
	}

	// rl := mw.NewRateLimiter(5, time.Minute)

	// hppOptions := mw.HPPOptions{
	// 	CheckQuery:                  true,
	// 	CheckBody:                   true,
	// 	CheckBodyOnlyForContentType: "application/x-www-form-urlencoded",
	// 	Whitelist:                   []string{"sortBy", "sortOrder", "name", "age", "class"},
	// }

	router := routers.MainRouter()
	jwtMiddleware := mw.MiddlewaresExcludePaths(mw.JWTMiddleware, "/users/signup", "/users/login", "/users/confirmotp", "/users/resendotp", "/users/forgotpassword", "/wallet/webhook")

	// secureMux := utils.ApplyMiddlewares(router, mw.SecurityHeaders, mw.Compression, mw.Hpp(hppOptions), jwtMiddleware, mw.ResponseTimeMiddleware, rl.Middleware, mw.Cors)
	secureMux := jwtMiddleware(mw.SecurityHeaders(router))

	server := &http.Server{
		Addr:      port,
		Handler:   secureMux,
		TLSConfig: tlsConfig,
	}

	fmt.Println("Server is running on port", port)
	err = server.ListenAndServeTLS(cert, key)
	if err != nil {
		log.Fatalln("Error starting the server", err)
	}

}
