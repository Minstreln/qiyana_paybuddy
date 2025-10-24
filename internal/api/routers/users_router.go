package routers

import (
	"net/http"
	"qiyana_paybuddy/internal/api/handlers/auth"
)

func usersRouter() *http.ServeMux {
	mux := http.NewServeMux()

	mux.HandleFunc("/users/signup", auth.RegisterUsersHandler)
	mux.HandleFunc("/users/confirmotp", auth.ConfirmOtpHandler)
	mux.HandleFunc("/users/resendotp", auth.ResendOtpHandler)

	mux.HandleFunc("/users/login", auth.LoginHandler)
	mux.HandleFunc("/users/logout", auth.LogoutHandler)
	mux.HandleFunc("/users/forgotpassword", auth.ForgotPasswordHandler)
	mux.HandleFunc("/users/resetpassword/reset/{resetcode}", auth.ResetPasswordHandler)
	mux.HandleFunc("/users/updatepassword", auth.UpdatePasswordHandler)

	return mux
}
