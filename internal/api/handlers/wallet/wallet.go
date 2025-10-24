package wallet

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"qiyana_paybuddy/internal/repositories/sqlconnect"
	"qiyana_paybuddy/internal/services"
	"qiyana_paybuddy/pkg/utils"
	"time"

	"github.com/shopspring/decimal"
)

func FundWallet(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		utils.WriteError(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	db := sqlconnect.DB
	if db == nil {
		utils.Logger.Error("DB is not initialized")
		utils.WriteError(w, "internal server error", http.StatusInternalServerError)
		return
	}

	idFloat, ok := r.Context().Value(utils.ContextKey("userId")).(float64)
	if !ok {
		utils.WriteError(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	userID := int(idFloat)

	type request struct {
		Amount      int    `json:"amount"`
		Description string `json:"description"`
	}

	var req request
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&req); err != nil {
		utils.WriteError(w, "enter amount", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	if req.Amount <= 0 {
		utils.WriteError(w, "amount must be greater than 0", http.StatusBadRequest)
		return
	}

	var email string
	err := db.QueryRow("SELECT email FROM users WHERE id = ?", userID).Scan(&email)
	if err != nil {
		if err == sql.ErrNoRows {
			utils.WriteError(w, "user not found", http.StatusNotFound)
			return
		}
		utils.Logger.Error("user not found")
		utils.WriteError(w, "user not found", http.StatusNotFound)
		return
	}

	username, _ := r.Context().Value(utils.ContextKey("username")).(string)

	paystack, err := services.NewPaystackClient()
	if err != nil {
		utils.WriteError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	amountKobo := req.Amount * 100
	description := req.Description

	form := map[string]interface{}{
		"email":  email,
		"amount": amountKobo,
		"metadata": map[string]interface{}{
			"userId":           userID,
			"transaction_type": "credit",
			"category":         "fund",
			"username":         username,
			"description":      description,
		},
	}

	res, err := paystack.InitializePayment(form)
	if err != nil {
		utils.Logger.Error("Payment initialization failed", "error", err, "user_id", userID)
		utils.WriteError(w, fmt.Sprintf("failed to initialize payment: %v", err), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(res)
}

// PaystackWebhook handles Paystack transaction notifications
func PaystackWebhook(w http.ResponseWriter, r *http.Request) {
	db := sqlconnect.DB
	if db == nil {
		utils.Logger.Error("DB is not initialized")
		utils.WriteError(w, "internal server error", http.StatusInternalServerError)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		utils.WriteError(w, "failed to read request body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	sig := r.Header.Get("X-Paystack-Signature")
	if !utils.VerifyPaystackSignature(sig, body) {
		utils.Logger.Warn("Invalid Paystack signature")
		utils.WriteError(w, "invalid signature", http.StatusUnauthorized)
		return
	}

	var payload struct {
		Event string `json:"event"`
		Data  struct {
			Reference string                 `json:"reference"`
			Amount    int                    `json:"amount"`
			Metadata  map[string]interface{} `json:"metadata"`
			Status    string                 `json:"status"`
		} `json:"data"`
	}

	if err := json.Unmarshal(body, &payload); err != nil {
		utils.WriteError(w, "invalid payload", http.StatusBadRequest)
		return
	}

	if payload.Event != "charge.success" || payload.Data.Status != "success" {
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ignored"))
		return
	}

	reference := payload.Data.Reference
	amountKobo := payload.Data.Amount
	amountNaira := amountKobo / 100

	transactionType, ok := payload.Data.Metadata["transaction_type"].(string)
	if !ok {
		utils.Logger.Error("Transaction type not found in metadata", "reference", reference)
		utils.WriteError(w, "invalid metadata", http.StatusBadRequest)
		return
	}

	category, ok := payload.Data.Metadata["category"].(string)
	if !ok {
		utils.Logger.Error("category not found in metadata", "reference", reference)
		utils.WriteError(w, "invalid metadata", http.StatusBadRequest)
		return
	}

	description, ok := payload.Data.Metadata["description"].(string)
	if !ok {
		utils.Logger.Error("description not found in metadata", "reference", reference)
		utils.WriteError(w, "invalid metadata", http.StatusBadRequest)
		return
	}

	var exists int
	err = db.QueryRow("SELECT COUNT(*) FROM transactions WHERE reference = ?", reference).Scan(&exists)
	if err != nil {
		utils.Logger.Error("Failed to check duplicate transaction", "error", err, "reference", reference)
		utils.WriteError(w, "database error", http.StatusInternalServerError)
		return
	}
	if exists > 0 {
		utils.Logger.Info("Duplicate transaction ignored", "reference", reference)
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
		return
	}

	var userID int
	switch v := payload.Data.Metadata["userId"].(type) {
	case float64:
		userID = int(v)
	case int:
		userID = v
	case string:
		fmt.Sscanf(v, "%d", &userID)
	default:
		utils.Logger.Error("User ID not found in metadata or invalid type", "reference", reference)
		utils.WriteError(w, "invalid metadata", http.StatusBadRequest)
		return
	}

	tx, err := db.Begin()
	if err != nil {
		utils.Logger.Error("Failed to start transaction", "error", err)
		utils.WriteError(w, "failed to start transaction", http.StatusInternalServerError)
		return
	}

	var walletExists int
	err = tx.QueryRow("SELECT COUNT(*) FROM wallets WHERE user_id = ?", userID).Scan(&walletExists)
	if err != nil {
		tx.Rollback()
		utils.Logger.Error("Failed to check wallet", "error", err, "user_id", userID)
		utils.WriteError(w, "database error", http.StatusInternalServerError)
		return
	}
	if walletExists == 0 {
		_, err = tx.Exec("INSERT INTO wallets (user_id, balance) VALUES (?, 0)", userID)
		if err != nil {
			tx.Rollback()
			utils.Logger.Error("Failed to create wallet", "error", err, "user_id", userID)
			utils.WriteError(w, "failed to create wallet", http.StatusInternalServerError)
			return
		}
	}

	amount := decimal.NewFromInt(int64(amountNaira))
	_, err = tx.Exec(`
	INSERT INTO transactions (user_id, transaction_type, category, amount, status, reference, description) 
	VALUES (?, ?, ?, ?, ?, ?, ?)`,
		userID, transactionType, category, amount, "success", reference, description)

	if err != nil {
		tx.Rollback()
		utils.Logger.Error("Failed to record transaction", "error", err, "reference", reference)
		utils.WriteError(w, "failed to record transaction", http.StatusInternalServerError)
		return
	}

	_, err = tx.Exec("UPDATE wallets SET balance = balance + ?, last_funded_at = ? WHERE user_id = ?", amountNaira, time.Now().Format("2006-01-02 15:04:05"), userID)
	if err != nil {
		tx.Rollback()
		utils.Logger.Error("Failed to update wallet", "error", err, "user_id", userID)
		utils.WriteError(w, "failed to update wallet", http.StatusInternalServerError)
		return
	}

	if err := tx.Commit(); err != nil {
		utils.Logger.Error("Failed to commit transaction", "error", err, "reference", reference)
		utils.WriteError(w, "failed to process payment", http.StatusInternalServerError)
		return
	}

	utils.Logger.Info("Transaction processed successfully", "reference", reference, "user_id", userID, "amount", amountNaira)

	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}
