package transactions

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"qiyana_paybuddy/internal/models"
	"qiyana_paybuddy/internal/repositories/sqlconnect"
	"qiyana_paybuddy/pkg/utils"
	"strconv"
	"time"
)

// FUNC TO GET ALL TRANSACTIONS FOR A USER
func GetAllUserTransactions(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		utils.WriteError(w, "Method Not Allowed", http.StatusMethodNotAllowed)
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

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	page, limit := utils.GetPaginationParams(r)
	offset := (page - 1) * limit

	query := `
		SELECT id, transaction_type, category, amount, status, reference, description, created_at, updated_at 
		FROM transactions
		WHERE user_id = ?
	`
	args := []interface{}{userID}

	query += " LIMIT ? OFFSET ?"
	args = append(args, limit, offset)

	query = utils.AddSorting(r, query)

	rows, err := db.QueryContext(ctx, query, args...)
	if err != nil {
		utils.Logger.Errorf("error fetching transactions: %v", err)
		utils.WriteError(w, "error fetching transactions", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var transactions []models.Transaction
	for rows.Next() {
		var transaction models.Transaction
		err = rows.Scan(&transaction.ID, &transaction.TransactionType, &transaction.Category, &transaction.Amount, &transaction.Status, &transaction.Reference, &transaction.Description, &transaction.CreatedAt, &transaction.UpdatedAt)
		if err != nil {
			utils.Logger.Errorf("error fetching data: %v", err)
			utils.WriteError(w, "error fetching transaction", http.StatusInternalServerError)
			return
		}
		transactions = append(transactions, transaction)
	}

	if len(transactions) == 0 {
		w.Header().Set("Content-Type", "application/json")
		response := map[string]interface{}{
			"status":  "success",
			"message": "no transaction found for this user",
			"data":    []models.Transaction{},
		}
		json.NewEncoder(w).Encode(response)
		return
	}

	response := struct {
		Status   string               `json:"status"`
		Count    int                  `json:"count"`
		Page     int                  `json:"page"`
		PageSize int                  `json:"page_size"`
		Data     []models.Transaction `json:"data"`
	}{
		Status:   "success",
		Count:    len(transactions),
		Page:     page,
		PageSize: limit,
		Data:     transactions,
	}

	utils.WriteJSON(w, response)
}

// FUNC TO GET ONE TRANSACTION BY ID
func GetTransactionById(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		utils.WriteError(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	idStr := r.PathValue("id")
	transactionID, err := strconv.Atoi(idStr)
	if err != nil {
		utils.WriteError(w, "invalid transaction ID", http.StatusBadRequest)
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

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	var transaction models.Transaction
	err = db.QueryRowContext(ctx, "SELECT transaction_type, category, amount, status, reference, description, created_at, updated_at FROM transactions WHERE id = ? AND user_id = ?", transactionID, userID).Scan(&transaction.TransactionType, &transaction.Category, &transaction.Amount, &transaction.Status, &transaction.Reference, &transaction.Description, &transaction.CreatedAt, &transaction.UpdatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			utils.WriteError(w, "no transaction found", http.StatusNotFound)
			return
		}
		utils.Logger.Errorf("error fetching data: %v", err)
		utils.WriteError(w, "error fetching transaction", http.StatusInternalServerError)
		return
	}

	response := struct {
		Status string             `json:"status"`
		Data   models.Transaction `json:"data"`
	}{
		Status: "success",
		Data:   transaction,
	}

	utils.WriteJSON(w, response)
}
