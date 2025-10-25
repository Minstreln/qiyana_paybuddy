package main

import (
	"context"
	"crypto/tls"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	mw "qiyana_paybuddy/internal/api/middlewares"
	"qiyana_paybuddy/internal/api/routers"
	"qiyana_paybuddy/internal/repositories/sqlconnect"
	"qiyana_paybuddy/pkg/cron"
	"qiyana_paybuddy/pkg/utils"

	"github.com/joho/godotenv"
)

func main() {
	if err := godotenv.Load(); err != nil {
		utils.Logger.Warn("No .env file found, using environment defaults")
	}

	utils.InitLogger()

	if err := sqlconnect.ConnectDb(); err != nil {
		utils.Logger.Fatal("DB connection failed: ", err)
	}
	db := sqlconnect.DB

	cronInstance := cron.StartCronJob(db)

	tlsConfig := &tls.Config{
		MinVersion: tls.VersionTLS10,
	}

	port := os.Getenv("SERVER_PORT")
	cert := os.Getenv("CERT_FILE")
	key := os.Getenv("KEY_FILE")

	rl := mw.NewRateLimiter(5, time.Minute)

	hppOptions := mw.HPPOptions{
		CheckQuery:                  true,
		CheckBody:                   true,
		CheckBodyOnlyForContentType: "application/x-www-form-urlencoded",
		Whitelist:                   []string{"sortBy", "sortOrder", "name", "age", "class"},
	}

	router := routers.MainRouter()

	jwtMiddleware := mw.MiddlewaresExcludePaths(
		mw.JWTMiddleware,
		"/api/v1/users/signup", "/api/v1/users/login", "/api/v1/users/confirmotp",
		"/api/v1/users/resendotp", "/api/v1/users/forgotpassword", "/api/v1/wallet/webhook",
	)

	secureMux := utils.ApplyMiddlewares(router, mw.SecurityHeaders, mw.Compression, mw.Hpp(hppOptions), jwtMiddleware, mw.ResponseTimeMiddleware, rl.Middleware, mw.Cors)

	// secureMux := jwtMiddleware(mw.SecurityHeaders(router))

	server := &http.Server{
		Addr:      port,
		Handler:   secureMux,
		TLSConfig: tlsConfig,
	}

	go func() {
		utils.Logger.Infof("Server running on port %s", port)
		if err := server.ListenAndServeTLS(cert, key); err != nil && err != http.ErrServerClosed {
			utils.Logger.Fatalf("Server error: %v", err)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)

	<-stop
	utils.Logger.Info("Shutting down server...")

	if cronInstance != nil {
		cronInstance.Stop()
		utils.Logger.Info("Cron job stopped.")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := server.Shutdown(ctx); err != nil {
		utils.Logger.Fatalf("Server forced to shutdown: %v", err)
	}

	utils.Logger.Info("Server exited gracefully ✅")
}
