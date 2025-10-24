package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"database/sql"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"qiyana_paybuddy/internal/api/handlers"
	"qiyana_paybuddy/internal/models"
	"qiyana_paybuddy/internal/repositories/sqlconnect"
	"qiyana_paybuddy/pkg/utils"
	"strconv"
	"strings"
	"time"

	"golang.org/x/crypto/argon2"
)

// FUNC TO REGISTER USERS
func RegisterUsersHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		utils.WriteError(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	db := sqlconnect.DB
	if db == nil {
		utils.Logger.Error("DB is not initialized")
		utils.WriteError(w, "internal server error", http.StatusInternalServerError)
		return
	}

	var newUser models.User
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&newUser); err != nil {
		utils.WriteError(w, "invalid or unexpected fields in body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	newUser.Role = "user"

	newUser.Username = strings.ToLower(newUser.Username)
	newUser.Email = strings.ToLower(newUser.Email)

	// Generate OTP and expiry
	duration, err := strconv.Atoi(os.Getenv("OTP_TOKEN_EXP_DURATION"))
	if err != nil {
		utils.Logger.Error("failed to read OTP_TOKEN_EXP_DURATION")
		utils.WriteError(w, "failed to generate otp", http.StatusInternalServerError)
		return
	}

	mins := time.Duration(duration)
	expiryTime := time.Now().Add(mins * time.Minute)
	expiryStr := expiryTime.Format(time.RFC3339)

	otp, err := utils.GenerateSecureOTP()
	if err != nil {
		utils.Logger.Errorf("failed to generate otp: %v", err)
		utils.WriteError(w, "failed to generate otp", http.StatusInternalServerError)
		return
	}

	newUser.Otp = otp
	newUser.OtpExpires = expiryStr

	if err := handlers.CheckBlankFields(newUser); err != nil {
		utils.WriteError(w, "missing required fields", http.StatusBadRequest)
		return
	}

	hashedPwd, err := utils.HashPassword(newUser.Password)
	if err != nil {
		utils.WriteError(w, "error hashing password", http.StatusInternalServerError)
		return
	}
	newUser.Password = hashedPwd

	tx, err := db.Begin()
	if err != nil {
		utils.Logger.Errorf("failed to start transaction: %v", err)
		utils.WriteError(w, "internal server error", http.StatusInternalServerError)
		return
	}

	stmt, err := tx.Prepare(utils.GenerateInsertQuery("users", models.User{}))
	if err != nil {
		tx.Rollback()
		utils.Logger.Errorf("failed to prepare statement: %v", err)
		utils.WriteError(w, "error signing up", http.StatusInternalServerError)
		return
	}
	defer stmt.Close()

	values := utils.GetStructValues(newUser)
	res, err := stmt.Exec(values...)
	if err != nil {
		tx.Rollback()
		if strings.Contains(err.Error(), "Duplicate entry") {
			utils.WriteError(w, "email or username already exists", http.StatusConflict)
			return
		}
		utils.Logger.Errorf("failed to insert user: %v", err)
		utils.WriteError(w, "error signing up", http.StatusInternalServerError)
		return
	}

	id, err := res.LastInsertId()
	if err != nil {
		tx.Rollback()
		utils.Logger.Errorf("failed to get last insert ID: %v", err)
		return
	}

	walletStmt, err := tx.Prepare(`
    INSERT INTO wallets (user_id, balance, last_funded_at)
    VALUES (?, 0.00, NULL)
	`)

	if err != nil {
		tx.Rollback()
		utils.Logger.Errorf("failed to prepare wallet statement: %v", err)
		utils.WriteError(w, "failed to create wallet", http.StatusInternalServerError)
		return
	}
	defer walletStmt.Close()

	if _, err := walletStmt.Exec(id); err != nil {
		tx.Rollback()
		utils.Logger.Errorf("failed to create wallet for user %d: %v", id, err)
		utils.WriteError(w, "failed to create wallet", http.StatusInternalServerError)
		return
	}

	if err = tx.Commit(); err != nil {
		tx.Rollback()
		utils.Logger.Errorf("failed to commit transaction: %v", err)
		utils.WriteError(w, "internal server error", http.StatusInternalServerError)
		return
	}

	go func(email, username, otp string, expiry time.Time) {
		if err := utils.SendOTPEmail(email, username, otp, expiry); err != nil {
			utils.Logger.Errorf("failed to send OTP email to %s: %v", email, err)
		}
	}(newUser.Email, newUser.Username, otp, expiryTime)

	newUser.ID = int(id)
	newUser.Password = ""

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  "success",
		"message": "OTP sent to your email for verification",
		"data":    newUser,
	})
}

// FUNC TO CONFIRM OTP
func ConfirmOtpHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		utils.WriteError(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	db := sqlconnect.DB
	if db == nil {
		utils.Logger.Error("DB is not initialized")
		utils.WriteError(w, "internal server error", http.StatusInternalServerError)
		return
	}

	type request struct {
		Otp string `json:"otp"`
	}

	var otp request
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&otp); err != nil {
		utils.WriteError(w, "invalid or unexpected fields in body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	if otp.Otp == "" {
		utils.WriteError(w, "please enter otp", http.StatusBadRequest)
		return
	}

	var user models.User
	query := "SELECT id, email, username FROM users WHERE otp = ? AND otp_expires > ?"
	err := db.QueryRow(query, otp.Otp, time.Now().Format(time.RFC3339)).Scan(&user.ID, &user.Email, &user.Username)
	if err != nil {
		utils.WriteError(w, "invalid or expired otp", http.StatusBadRequest)
		return
	}

	go func(email, username string) {
		if err := utils.SendWelcomeEmail(email, username); err != nil {
			utils.Logger.Errorf("failed to send OTP email to %s: %v", email, err)
		}
	}(user.Email, user.Username)

	updateQuery := "UPDATE users SET otp = NULL, otp_expires = NULL, email_confirmed = ? WHERE id = ?"

	_, err = db.Exec(updateQuery, true, user.ID)
	if err != nil {
		utils.WriteError(w, "could not verify otp", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  "success",
		"message": "OTP verified successfully, Welcome onboard!",
	})
}

// FUNC TO RESEND OTP
func ResendOtpHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		utils.WriteError(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	db := sqlconnect.DB
	if db == nil {
		utils.Logger.Error("DB is not initialized")
		utils.WriteError(w, "internal server error", http.StatusInternalServerError)
		return
	}

	type request struct {
		Email string `json:"email"`
	}

	var req request
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&req); err != nil {
		utils.WriteError(w, "invalid or unexpected fields in body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	req.Email = strings.ToLower(req.Email)
	if req.Email == "" {
		utils.WriteError(w, "please enter your email", http.StatusBadRequest)
		return
	}

	var user models.User
	query := "SELECT id, email, username, email_confirmed FROM users WHERE email = ?"
	err := db.QueryRow(query, req.Email).Scan(&user.ID, &user.Email, &user.Username, &user.EmailConfirmed)
	if err != nil {
		utils.WriteError(w, "user not found", http.StatusNotFound)
		return
	}

	if user.EmailConfirmed {
		utils.WriteError(w, "email already verified", http.StatusBadRequest)
		return
	}

	// Generate OTP and expiry
	duration, err := strconv.Atoi(os.Getenv("OTP_TOKEN_EXP_DURATION"))
	if err != nil {
		utils.Logger.Error("failed to read OTP_TOKEN_EXP_DURATION")
		utils.WriteError(w, "failed to generate otp", http.StatusInternalServerError)
		return
	}

	mins := time.Duration(duration)
	expiryTime := time.Now().Add(mins * time.Minute)
	expiryStr := expiryTime.Format(time.RFC3339)

	otp, err := utils.GenerateSecureOTP()
	if err != nil {
		utils.Logger.Errorf("failed to generate otp: %v", err)
		utils.WriteError(w, "failed to generate otp", http.StatusInternalServerError)
		return
	}

	updatedQuery := "UPDATE users SET otp = ?, otp_expires = ? WHERE id = ?"
	_, err = db.Exec(updatedQuery, otp, expiryStr, user.ID)
	if err != nil {
		utils.Logger.Errorf("failed to update user otp: %v", err)
		utils.WriteError(w, "could not update otp", http.StatusInternalServerError)
		return
	}

	go func(email, username, otp string, expiry time.Time) {
		if err := utils.SendOTPEmail(email, username, otp, expiry); err != nil {
			utils.Logger.Errorf("failed to send OTP email to %s: %v", email, err)
		}
	}(user.Email, user.Username, otp, expiryTime)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  "success",
		"message": "New OTP sent to your email successfully",
	})
}

// FUNC TO LOGIN
func LoginHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		utils.WriteError(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	db := sqlconnect.DB
	if db == nil {
		utils.Logger.Error("DB is not initialized")
		utils.WriteError(w, "internal server error", http.StatusInternalServerError)
		return
	}

	type loginRequest struct {
		AccountID string `json:"account_id"`
		Password  string `json:"password"`
	}

	var req loginRequest

	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		utils.WriteError(w, "invalid request body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	if req.AccountID == "" || req.Password == "" {
		utils.WriteError(w, "email or username and password are required", http.StatusBadRequest)
		return
	}

	user := &models.User{}

	query := "SELECT id, first_name, last_name, email, username, password, inactive_status, role FROM users WHERE username = ? OR email = ?"
	err = db.QueryRow(query, req.AccountID, req.AccountID).Scan(&user.ID, &user.FirstName, &user.LastName, &user.Email, &user.Username, &user.Password, &user.InactiveStatus, &user.Role)
	if err != nil {
		if err == sql.ErrNoRows {
			utils.WriteError(w, "user not found", http.StatusNotFound)
			utils.Logger.Error("user not found")
			return
		}
		utils.Logger.Error("database query error")
		utils.WriteError(w, "internal error", http.StatusInternalServerError)
		return
	}

	if user.InactiveStatus {
		utils.WriteError(w, "user account is not active", http.StatusForbidden)
		return
	}

	parts := strings.Split(user.Password, ".")
	if len(parts) != 2 {
		utils.Logger.Error("invalid encoded hash format")
		utils.WriteError(w, "invalid password", http.StatusForbidden)
		return
	}

	saltBase64 := parts[0]
	hashedPasswordBase64 := parts[1]

	salt, err := base64.StdEncoding.DecodeString(saltBase64)
	if err != nil {
		utils.Logger.Error("failed to decode salt")
		utils.WriteError(w, "invalid password", http.StatusForbidden)
		return
	}

	hashPassword, err := base64.StdEncoding.DecodeString(hashedPasswordBase64)
	if err != nil {
		utils.Logger.Error("failed to decode hashed password")
		utils.WriteError(w, "invalid password", http.StatusForbidden)
		return
	}

	hash := argon2.IDKey([]byte(req.Password), salt, 1, 64*1024, 4, 32)
	if len(hash) != len(hashPassword) {
		utils.WriteError(w, "incorrect password or account ID", http.StatusForbidden)
		return
	}

	if subtle.ConstantTimeCompare(hash, hashPassword) == 1 {
	} else {
		utils.WriteError(w, "incorrect password or account ID", http.StatusForbidden)
		return
	}

	tokenString, err := utils.SignToken(user.ID, user.Username, user.Role)
	if err != nil {
		utils.Logger.Error("could not create login token")
		utils.WriteError(w, "error signing in", http.StatusInternalServerError)
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "Bearer",
		Value:    tokenString,
		Path:     "/",
		HttpOnly: true,
		Secure:   true,
		Expires:  time.Now().Add(24 * time.Hour),
		SameSite: http.SameSiteStrictMode,
	})

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	response := map[string]interface{}{
		"status":  "success",
		"message": "login successful",
		"token":   tokenString,
		"user": map[string]interface{}{
			"id":        user.ID,
			"firstName": user.FirstName,
			"lastName":  user.LastName,
			"email":     user.Email,
			"username":  user.Username,
			"role":      user.Role,
		},
	}

	json.NewEncoder(w).Encode(response)
}

// FUNC FOR LOGOUT
func LogoutHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		utils.WriteError(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "Bearer",
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		Secure:   true,
		Expires:  time.Unix(0, 0),
		SameSite: http.SameSiteStrictMode,
	})

	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"message": "logged out successfully"}`))
}

// FUNC TO UPDATE PASSWORD
func UpdatePasswordHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPatch {
		utils.WriteError(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	idFloat, ok := r.Context().Value(utils.ContextKey("userId")).(float64)
	if !ok {
		utils.WriteError(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	userID := int(idFloat)

	db := sqlconnect.DB
	if db == nil {
		utils.Logger.Error("DB is not initialized")
		utils.WriteError(w, "internal server error", http.StatusInternalServerError)
		return
	}

	var req models.UpdatePasswordRequest
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&req); err != nil {
		utils.WriteError(w, "all fields are required", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	if req.CurrentPassword == "" || req.NewPassword == "" {
		utils.WriteError(w, "please enter all fields", http.StatusBadRequest)
		return
	}

	var userRole string
	var username string
	var userPassword string

	err := db.QueryRow("SELECT password, username, role FROM users WHERE id = ?", userID).Scan(&userPassword, &username, &userRole)
	if err != nil {
		utils.WriteError(w, "user not found", http.StatusNotFound)
		return
	}

	err = utils.VerifyPassword(req.CurrentPassword, userPassword)
	if err != nil {
		utils.WriteError(w, "the password you entered does not match the current password", http.StatusBadRequest)
		return
	}

	hashedPassword, err := utils.HashPassword(req.NewPassword)
	if err != nil {
		utils.Logger.Error("failed to hash password")
		utils.WriteError(w, "internal server error", http.StatusInternalServerError)
		return
	}

	currentTime := time.Now().Format(time.RFC3339)

	_, err = db.Exec("UPDATE users SET password = ?, password_changed_at = ? WHERE id = ?", hashedPassword, currentTime, userID)
	if err != nil {
		utils.WriteError(w, "failed to update password", http.StatusInternalServerError)
		return
	}

	token, err := utils.SignToken(userID, username, userRole)
	if err != nil {
		utils.Logger.Error("could not create token")
		utils.WriteError(w, "internal server error", http.StatusInternalServerError)
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "Bearer",
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		Secure:   true,
		Expires:  time.Now().Add(24 * time.Hour),
		SameSite: http.SameSiteStrictMode,
	})

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	response := map[string]interface{}{
		"status":  "success",
		"message": "password updated successfully",
	}

	json.NewEncoder(w).Encode(response)
}

// FUNC FOR FORGOT PASSWORD
func ForgotPasswordHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		utils.WriteError(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	db := sqlconnect.DB
	if db == nil {
		utils.Logger.Error("DB is not initialized")
		utils.WriteError(w, "internal server error", http.StatusInternalServerError)
		return
	}

	var req struct {
		Email string `json:"email"`
	}

	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&req); err != nil {
		utils.WriteError(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if req.Email == "" {
		utils.WriteError(w, "please enter email", http.StatusBadRequest)
		return
	}

	var user models.User
	err := db.QueryRow("SELECT id, username FROM users WHERE email = ?", req.Email).Scan(&user.ID, &user.Username)
	if err != nil {
		utils.WriteError(w, "user not found", http.StatusNotFound)
		return
	}

	duration, err := strconv.Atoi(os.Getenv("RESET_TOKEN_EXP_DURATION"))
	if err != nil {
		utils.ErrorHandler(err, "failed to send password reset email")
		return
	}

	mins := time.Duration(duration)

	expiryTime := time.Now().Add(mins * time.Minute)
	expiry := expiryTime.Format(time.RFC3339)

	tokenBytes := make([]byte, 32)
	_, err = rand.Read(tokenBytes)
	if err != nil {
		utils.ErrorHandler(err, "failed to send password reset email")
		return
	}

	token := hex.EncodeToString(tokenBytes)

	hashedToken := sha256.Sum256(tokenBytes)

	hashedTokenString := hex.EncodeToString(hashedToken[:])

	_, err = db.Exec("UPDATE users SET password_reset_token = ?, password_token_expires = ? WHERE id = ?", hashedTokenString, expiry, user.ID)
	if err != nil {
		utils.Logger.Error("failed to send password reset email")
		utils.WriteError(w, "failed to send reset email", http.StatusInternalServerError)
		return
	}

	resetUrl := fmt.Sprintf("https://localhost:3000/users/resetpassword/reset/%s", token)

	go func(email, username, resetURL string, expiry time.Time) {
		if err := utils.SendPasswordResetEmail(email, username, resetURL, expiry); err != nil {
			utils.Logger.Errorf("failed to send OTP email to %s: %v", email, err)
		}
	}(req.Email, user.Username, resetUrl, expiryTime)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  "success",
		"message": "password reset token sent to email",
	})
}

// FUNC TO RESET PASSWORD
func ResetPasswordHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPatch {
		utils.WriteError(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	db := sqlconnect.DB
	if db == nil {
		utils.Logger.Error("DB is not initialized")
		utils.WriteError(w, "internal server error", http.StatusInternalServerError)
		return
	}

	token := r.PathValue("resetcode")

	type request struct {
		NewPassword     string `json:"new_password"`
		ConfirmPassword string `json:"confirm_password"`
	}

	var req request

	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		utils.WriteError(w, "invalid values in request", http.StatusBadRequest)
		return
	}

	if req.NewPassword == "" || req.ConfirmPassword == "" {
		utils.WriteError(w, "All fields are required", http.StatusBadRequest)
		return
	}

	if req.NewPassword != req.ConfirmPassword {
		utils.WriteError(w, "Passwords should match", http.StatusBadRequest)
		return
	}

	bytes, err := hex.DecodeString(token)
	if err != nil {
		utils.ErrorHandler(err, "internal error")
		return
	}

	hashedToken := sha256.Sum256(bytes)
	hashedTokenString := hex.EncodeToString(hashedToken[:])

	var user models.User

	query := "SELECT id, email FROM users WHERE password_reset_token = ? AND password_token_expires > ?"
	err = db.QueryRow(query, hashedTokenString, time.Now().Format(time.RFC3339)).Scan(&user.ID, &user.Email)
	if err != nil {
		utils.WriteError(w, "invalid or expired reset code", http.StatusBadRequest)
		return
	}

	hashedPassword, err := utils.HashPassword(req.NewPassword)
	if err != nil {
		utils.ErrorHandler(err, "internal error")
		return
	}

	updateQuery := "UPDATE users SET password = ?, password_reset_token = NULL, password_token_expires = NULL, password_changed_at = ? WHERE id = ?"
	_, err = db.Exec(updateQuery, hashedPassword, time.Now().Format(time.RFC3339), user.ID)
	if err != nil {
		utils.Logger.Error("Could not update password")
		utils.WriteError(w, "could not update password", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  "success",
		"message": "password reset successfully",
	})
}
