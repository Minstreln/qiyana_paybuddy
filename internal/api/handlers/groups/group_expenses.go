package groups

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"qiyana_paybuddy/internal/models"
	"qiyana_paybuddy/internal/repositories/sqlconnect"
	"qiyana_paybuddy/pkg/utils"
	"reflect"
	"strconv"
	"time"

	"github.com/shopspring/decimal"
)

// FUNC TO CREATE GROUP EXPENSES
func CreateGroupExpenseHandler(w http.ResponseWriter, r *http.Request) {
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
		GroupID     int             `json:"group_id"`
		Description string          `json:"description"`
		Amount      decimal.Decimal `json:"amount"`
	}

	var req request
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&req); err != nil {
		utils.WriteError(w, "invalid request body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	if req.Amount.LessThanOrEqual(decimal.Zero) {
		utils.WriteError(w, "amount must be greater than 0", http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	var group models.Group
	err := db.QueryRowContext(ctx, "SELECT name, description, created_by FROM groups WHERE id = ?", req.GroupID).
		Scan(&group.Name, &group.Description, &group.CreatedBy)
	if err != nil {
		if err == sql.ErrNoRows {
			utils.WriteError(w, "group not found", http.StatusNotFound)
			return
		}
		utils.WriteError(w, "failed to retrieve group", http.StatusInternalServerError)
		return
	}

	var exists bool
	err = db.QueryRowContext(ctx, "SELECT EXISTS(SELECT 1 FROM group_members WHERE group_id = ? AND user_id = ?)", req.GroupID, userID).Scan(&exists)
	if err != nil {
		utils.WriteError(w, "failed to verify group membership", http.StatusInternalServerError)
		return
	}
	if !exists {
		utils.WriteError(w, "you are not a member of this group", http.StatusForbidden)
		return
	}

	rows, err := db.QueryContext(ctx, "SELECT user_id FROM group_members WHERE group_id = ? AND user_id != ?", req.GroupID, userID)
	if err != nil {
		utils.WriteError(w, "failed to fetch group members", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var memberIDs []int
	for rows.Next() {
		var memberID int
		if err := rows.Scan(&memberID); err == nil {
			memberIDs = append(memberIDs, memberID)
		}
	}

	if len(memberIDs) == 0 {
		utils.WriteError(w, "no members to split expense with", http.StatusBadRequest)
		return
	}

	totalMembers := len(memberIDs) + 1
	share := req.Amount.Div(decimal.NewFromInt(int64(totalMembers))).Round(2)

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		utils.Logger.Errorf("failed to start transaction: %v", err)
		utils.WriteError(w, "failed to start transaction", http.StatusInternalServerError)
		return
	}

	res, err := tx.ExecContext(ctx, "INSERT INTO group_expenses (group_id, paid_by, description, amount, created_at) VALUES (?, ?, ?, ?, ?)",
		req.GroupID, userID, req.Description, req.Amount, time.Now().Format("2006-01-02 15:04:05"))
	if err != nil {
		tx.Rollback()
		utils.WriteError(w, "failed to create expense", http.StatusInternalServerError)
		return
	}

	expenseID, _ := res.LastInsertId()

	stmt, err := tx.PrepareContext(ctx, `INSERT INTO group_expense_splits (expense_id, owed_by, amount_owed, is_settled) VALUES (?, ?, ?, FALSE)`)
	if err != nil {
		tx.Rollback()
		utils.Logger.Errorf("failed to prepare statement: %v", err)
		utils.WriteError(w, "internal server error", http.StatusInternalServerError)
		return
	}
	defer stmt.Close()

	for _, memberID := range memberIDs {
		if _, err := stmt.ExecContext(ctx, expenseID, memberID, share); err != nil {
			tx.Rollback()
			utils.Logger.Errorf("failed to split expense: %v", err)
			utils.WriteError(w, "failed to split expense", http.StatusInternalServerError)
			return
		}
	}

	if err := tx.Commit(); err != nil {
		utils.WriteError(w, "failed to commit transaction", http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"status":  "success",
		"message": fmt.Sprintf("Expense created and split among %d members (including payer)", totalMembers),
		"data": map[string]interface{}{
			"expense_id": expenseID,
			"amount":     req.Amount,
			"split_each": share,
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// FUNC TO GET ALL GROUP EXPENSES
func GetGroupExpensesHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		utils.WriteError(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	db := sqlconnect.DB
	if db == nil {
		utils.Logger.Error("DB is not initialized")
		utils.WriteError(w, "internal server error", http.StatusInternalServerError)
		return
	}

	idStr := r.PathValue("id")
	groupID, err := strconv.Atoi(idStr)
	if err != nil {
		utils.WriteError(w, "invalid group ID", http.StatusBadRequest)
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

	var group models.Group
	err = db.QueryRowContext(ctx, "SELECT name, description, created_by FROM groups WHERE id = ?", groupID).
		Scan(&group.Name, &group.Description, &group.CreatedBy)
	if err != nil {
		if err == sql.ErrNoRows {
			utils.WriteError(w, "group not found", http.StatusNotFound)
			return
		}
		utils.WriteError(w, "failed to retrieve group", http.StatusInternalServerError)
		return
	}

	var exists bool
	err = db.QueryRowContext(ctx, "SELECT EXISTS(SELECT 1 FROM group_members WHERE group_id = ? AND user_id = ?)", groupID, userID).Scan(&exists)
	if err != nil {
		utils.WriteError(w, "failed to verify group membership", http.StatusInternalServerError)
		return
	}
	if !exists {
		utils.WriteError(w, "you are not a member of this group", http.StatusForbidden)
		return
	}

	query := `
		SELECT e.id, e.description, e.amount, u.username AS paid_by, e.created_at
		FROM group_expenses e
		JOIN users u ON e.paid_by = u.id
		WHERE e.group_id = ?
		ORDER BY e.created_at DESC
	`
	rows, err := db.QueryContext(ctx, query, groupID)
	if err != nil {
		utils.WriteError(w, "failed to retrieve expenses", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	type Expense struct {
		ID          int            `json:"id"`
		Description string         `json:"description"`
		Amount      float64        `json:"amount"`
		PaidBy      string         `json:"paid_by"`
		CreatedAt   sql.NullString `json:"created_at"`
	}

	var expenses []Expense

	for rows.Next() {
		var e Expense
		err := rows.Scan(&e.ID, &e.Description, &e.Amount, &e.PaidBy, &e.CreatedAt)
		if err != nil {
			utils.Logger.Errorf("error reading expenses: %v", err)
			utils.WriteError(w, "error reading expenses", http.StatusInternalServerError)
			return
		}
		expenses = append(expenses, e)
	}

	if err = rows.Err(); err != nil {
		utils.WriteError(w, "error finalizing expenses read", http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"status":   "success",
		"group_id": groupID,
		"count":    len(expenses),
		"expenses": expenses,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// FUNC TO GET ONE EXPENSE DETAILS
func GetExpenseByIdHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		utils.WriteError(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	db := sqlconnect.DB
	if db == nil {
		utils.Logger.Error("DB is not initialized")
		utils.WriteError(w, "internal server error", http.StatusInternalServerError)
		return
	}

	idStr := r.PathValue("id")
	expenseID, err := strconv.Atoi(idStr)
	if err != nil {
		utils.WriteError(w, "invalid expense ID", http.StatusBadRequest)
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

	var expense models.GroupExpense
	err = db.QueryRowContext(ctx, "SELECT group_id, paid_by, description, amount FROM group_expenses WHERE id = ?", expenseID).
		Scan(&expense.GroupID, &expense.PaidBy, &expense.Description, &expense.Amount)
	if err != nil {
		if err == sql.ErrNoRows {
			utils.WriteError(w, "expense not found", http.StatusNotFound)
			return
		}
		utils.WriteError(w, "failed to retrieve expense", http.StatusInternalServerError)
		return
	}

	var group models.Group
	err = db.QueryRowContext(ctx, "SELECT name, description, created_by FROM groups WHERE id = ?", expense.GroupID).
		Scan(&group.Name, &group.Description, &group.CreatedBy)
	if err != nil {
		if err == sql.ErrNoRows {
			utils.WriteError(w, "group not found", http.StatusNotFound)
			return
		}
		utils.WriteError(w, "failed to retrieve group", http.StatusInternalServerError)
		return
	}

	var exists bool
	err = db.QueryRowContext(ctx, "SELECT EXISTS(SELECT 1 FROM group_members WHERE group_id = ? AND user_id = ?)", expense.GroupID, userID).Scan(&exists)
	if err != nil {
		utils.WriteError(w, "failed to verify group membership", http.StatusInternalServerError)
		return
	}
	if !exists {
		utils.WriteError(w, "you are not a member of this group", http.StatusForbidden)
		return
	}

	type GroupExpenseSplit struct {
		ID         int             `json:"id"`
		OwedBy     int             `json:"owed_by"`
		Username   string          `json:"username"`
		AmountOwed decimal.Decimal `json:"amount_owed"`
		IsSettled  bool            `json:"is_settled"`
	}

	query := `
		SELECT s.id, s.owed_by, u.username, s.amount_owed, s.is_settled
		FROM group_expense_splits s
		JOIN users u ON s.owed_by = u.id
		WHERE s.expense_id = ?;
	`
	rows, err := db.QueryContext(ctx, query, expenseID)
	if err != nil {
		utils.Logger.Errorf("failed to retrieve expense splits: %v", err)
		utils.WriteError(w, "failed to retrieve expense splits", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var splits []GroupExpenseSplit
	for rows.Next() {
		var s GroupExpenseSplit
		if err := rows.Scan(&s.ID, &s.OwedBy, &s.Username, &s.AmountOwed, &s.IsSettled); err != nil {
			utils.Logger.Errorf("error scanning split: %v", err)
			continue
		}
		splits = append(splits, s)
	}

	response := map[string]interface{}{
		"status": "success",
		"data": map[string]interface{}{
			"expense": map[string]interface{}{
				"description": expense.Description,
				"amount":      expense.Amount,
				"paid_by":     expense.PaidBy,
				"group_id":    expense.GroupID,
			},
			"group": map[string]interface{}{
				"name":        group.Name,
				"description": group.Description,
				"created_by":  group.CreatedBy,
			},
			"splits": splits,
		},
	}

	utils.WriteJSON(w, response)
}

// FUNC TO UPDATE GROUP EXPENSES
func UpdateGroupExpensesHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPatch {
		utils.WriteError(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	db := sqlconnect.DB
	if db == nil {
		utils.Logger.Error("DB is not initialized")
		utils.WriteError(w, "internal server error", http.StatusInternalServerError)
		return
	}

	idStr := r.PathValue("id")
	expenseID, err := strconv.Atoi(idStr)
	if err != nil {
		utils.WriteError(w, "invalid expense ID", http.StatusBadRequest)
		return
	}

	idFloat, ok := r.Context().Value(utils.ContextKey("userId")).(float64)
	if !ok {
		utils.WriteError(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	userID := int(idFloat)

	var request map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		utils.WriteError(w, "invalid request payload", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	var expense models.GroupExpense
	err = db.QueryRowContext(ctx, "SELECT id, group_id, paid_by, description, amount FROM group_expenses WHERE id = ?", expenseID).
		Scan(&expense.ID, &expense.GroupID, &expense.PaidBy, &expense.Description, &expense.Amount)
	if err != nil {
		if err == sql.ErrNoRows {
			utils.WriteError(w, "expense not found", http.StatusNotFound)
			return
		}
		utils.WriteError(w, "failed to retrieve expense", http.StatusInternalServerError)
		return
	}

	if expense.PaidBy != userID {
		utils.WriteError(w, "you are not authorized to edit this expense entry", http.StatusUnauthorized)
		return
	}

	var exists bool
	err = db.QueryRowContext(ctx, "SELECT EXISTS(SELECT 1 FROM group_members WHERE group_id = ? AND user_id = ?)", expense.GroupID, userID).Scan(&exists)
	if err != nil {
		utils.WriteError(w, "failed to verify group membership", http.StatusInternalServerError)
		return
	}
	if !exists {
		utils.WriteError(w, "you are not a member of this group", http.StatusForbidden)
		return
	}

	if amountVal, ok := request["amount"]; ok {
		var newAmount decimal.Decimal
		switch v := amountVal.(type) {
		case float64:
			newAmount = decimal.NewFromFloat(v)
		case string:
			newAmount, err = decimal.NewFromString(v)
			if err != nil {
				utils.WriteError(w, "invalid amount format", http.StatusBadRequest)
				return
			}
		default:
			utils.WriteError(w, "invalid amount type", http.StatusBadRequest)
			return
		}
		request["amount"] = newAmount
	}

	expenseVal := reflect.ValueOf(&expense).Elem()
	expenseType := expenseVal.Type()

	for k, v := range request {
		for i := 0; i < expenseVal.NumField(); i++ {
			field := expenseType.Field(i)
			jsonTag := field.Tag.Get("json")
			if jsonTag == k || jsonTag == k+",omitempty" {
				if expenseVal.Field(i).CanSet() {
					val := reflect.ValueOf(v)
					if val.Type().ConvertibleTo(expenseVal.Field(i).Type()) {
						expenseVal.Field(i).Set(val.Convert(expenseVal.Field(i).Type()))
					}
				}
			}
		}
	}

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		utils.Logger.Errorf("failed to start transaction: %v", err)
		utils.WriteError(w, "internal server error", http.StatusInternalServerError)
		return
	}

	_, err = tx.ExecContext(ctx,
		"UPDATE group_expenses SET description = ?, amount = ? WHERE id = ?",
		expense.Description, expense.Amount, expense.ID)
	if err != nil {
		tx.Rollback()
		utils.WriteError(w, "error updating expense", http.StatusInternalServerError)
		return
	}

	rows, err := tx.QueryContext(ctx, "SELECT user_id FROM group_members WHERE group_id = ? AND user_id != ?", expense.GroupID, userID)
	if err != nil {
		tx.Rollback()
		utils.WriteError(w, "failed to fetch group members", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var memberIDs []int
	for rows.Next() {
		var memberID int
		if err := rows.Scan(&memberID); err == nil {
			memberIDs = append(memberIDs, memberID)
		}
	}

	totalMembers := len(memberIDs) + 1
	if totalMembers == 0 {
		tx.Rollback()
		utils.WriteError(w, "no members found to split expense", http.StatusBadRequest)
		return
	}

	share := expense.Amount.Div(decimal.NewFromInt(int64(totalMembers))).Round(2)

	_, err = tx.ExecContext(ctx, "DELETE FROM group_expense_splits WHERE expense_id = ?", expense.ID)
	if err != nil {
		tx.Rollback()
		utils.WriteError(w, "failed to reset splits", http.StatusInternalServerError)
		return
	}

	stmt, err := tx.PrepareContext(ctx, `INSERT INTO group_expense_splits (expense_id, owed_by, amount_owed, is_settled) VALUES (?, ?, ?, FALSE)`)
	if err != nil {
		tx.Rollback()
		utils.WriteError(w, "failed to prepare statement", http.StatusInternalServerError)
		return
	}
	defer stmt.Close()

	for _, memberID := range memberIDs {
		if _, err := stmt.ExecContext(ctx, expense.ID, memberID, share); err != nil {
			tx.Rollback()
			utils.WriteError(w, "failed to recreate splits", http.StatusInternalServerError)
			return
		}
	}

	if err := tx.Commit(); err != nil {
		tx.Rollback()
		utils.WriteError(w, "failed to commit transaction", http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"status":  "success",
		"message": "Expense updated successfully",
		"data": map[string]interface{}{
			"expense_id": expense.ID,
			"new_amount": expense.Amount,
			"split_each": share,
		},
	}

	utils.WriteJSON(w, response)
}

// FUNC TO GET WHAT A USER OWES / IS OWED
func GetUserBalanceSummaryHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
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

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	type BalanceSummary struct {
		UserID    int             `json:"user_id"`
		Username  string          `json:"username"`
		SplitID   int             `json:"split_id"`
		TotalOwed decimal.Decimal `json:"total_owed"`
	}

	owesQuery := `
		SELECT e.paid_by, u.username, s.id, SUM(s.amount_owed) AS total_owed
		FROM group_expense_splits s
		JOIN group_expenses e ON s.expense_id = e.id
		JOIN users u ON e.paid_by = u.id
		WHERE s.owed_by = ? AND s.is_settled = FALSE
		GROUP BY e.paid_by, u.username;
	`

	rows1, err := db.QueryContext(ctx, owesQuery, userID)
	if err != nil {
		utils.WriteError(w, "failed to fetch owed summary", http.StatusInternalServerError)
		return
	}
	defer rows1.Close()

	var owes []BalanceSummary
	for rows1.Next() {
		var b BalanceSummary
		if err := rows1.Scan(&b.UserID, &b.Username, &b.SplitID, &b.TotalOwed); err != nil {
			utils.Logger.Errorf("error scanning owes summary: %v", err)
			continue
		}
		owes = append(owes, b)
	}

	isOwedQuery := `
		SELECT s.owed_by, u.username, s.id, SUM(s.amount_owed) AS total_owed
		FROM group_expense_splits s
		JOIN group_expenses e ON s.expense_id = e.id
		JOIN users u ON s.owed_by = u.id
		WHERE e.paid_by = ? AND s.is_settled = FALSE
		GROUP BY s.owed_by, u.username;
	`

	rows2, err := db.QueryContext(ctx, isOwedQuery, userID)
	if err != nil {
		utils.WriteError(w, "failed to fetch is_owed summary", http.StatusInternalServerError)
		return
	}
	defer rows2.Close()

	var isOwed []BalanceSummary
	for rows2.Next() {
		var b BalanceSummary
		if err := rows2.Scan(&b.UserID, &b.Username, &b.SplitID, &b.TotalOwed); err != nil {
			utils.Logger.Errorf("error scanning is_owed summary: %v", err)
			continue
		}
		isOwed = append(isOwed, b)
	}

	response := map[string]interface{}{
		"status": "success",
		"data": map[string]interface{}{
			"owes":    owes,
			"is_owed": isOwed,
		},
	}

	utils.WriteJSON(w, response)
}

// FUNC TO GET GROUP SUMMARY
func GetGroupSummaryHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		utils.WriteError(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	db := sqlconnect.DB
	if db == nil {
		utils.WriteError(w, "internal server error", http.StatusInternalServerError)
		return
	}

	idStr := r.PathValue("id")
	groupID, err := strconv.Atoi(idStr)
	if err != nil {
		utils.WriteError(w, "invalid group ID", http.StatusBadRequest)
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

	var groupName string
	err = db.QueryRowContext(ctx, "SELECT name FROM groups WHERE id = ?", groupID).Scan(&groupName)
	if err != nil {
		if err == sql.ErrNoRows {
			utils.WriteError(w, "group not found", http.StatusNotFound)
			return
		}
		utils.WriteError(w, "failed to fetch group", http.StatusInternalServerError)
		return
	}

	var isMember bool
	err = db.QueryRowContext(ctx, "SELECT EXISTS(SELECT 1 FROM group_members WHERE group_id = ? AND user_id = ?)", groupID, userID).Scan(&isMember)
	if err != nil {
		utils.WriteError(w, "failed to verify group membership", http.StatusInternalServerError)
		return
	}
	if !isMember {
		utils.WriteError(w, "you are not a member of this group", http.StatusForbidden)
		return
	}

	type GroupBalance struct {
		UserID    int             `json:"user_id"`
		Username  string          `json:"username"`
		TotalOwed decimal.Decimal `json:"total_owed"`
	}

	query := `
		SELECT s.owed_by, u.username, SUM(s.amount_owed) AS total_owed
		FROM group_expense_splits s
		JOIN group_expenses e ON e.id = s.expense_id
		JOIN users u ON s.owed_by = u.id
		WHERE e.group_id = ?
		GROUP BY s.owed_by, u.username;
	`

	rows, err := db.QueryContext(ctx, query, groupID)
	if err != nil {
		utils.WriteError(w, "failed to fetch group summary", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var balances []GroupBalance
	for rows.Next() {
		var gb GroupBalance
		if err := rows.Scan(&gb.UserID, &gb.Username, &gb.TotalOwed); err != nil {
			utils.Logger.Errorf("error scanning group balance: %v", err)
			continue
		}
		balances = append(balances, gb)
	}

	response := map[string]interface{}{
		"status": "success",
		"data": map[string]interface{}{
			"group": map[string]interface{}{
				"id":   groupID,
				"name": groupName,
			},
			"balances": balances,
		},
	}

	utils.WriteJSON(w, response)
}

// FUNC TO SETTLE EXPENSE SPLIT
func SettleExpenseSplitHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		utils.WriteError(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	db := sqlconnect.DB
	if db == nil {
		utils.WriteError(w, "internal server error", http.StatusInternalServerError)
		return
	}

	idStr := r.PathValue("split_id")
	splitID, err := strconv.Atoi(idStr)
	if err != nil {
		utils.WriteError(w, "invalid group ID", http.StatusBadRequest)
		return
	}

	idFloat, ok := r.Context().Value(utils.ContextKey("userId")).(float64)
	if !ok {
		utils.WriteError(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	userID := int(idFloat)

	type request struct {
		Amount decimal.Decimal `json:"amount"`
	}

	var req request
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&req); err != nil {
		utils.WriteError(w, "invalid request body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	ctx, cancel := context.WithTimeout(r.Context(), 20*time.Second)
	defer cancel()

	var split models.GroupExpenseSplit
	err = db.QueryRowContext(ctx, "SELECT id, expense_id, owed_by, amount_owed, created_at FROM group_expense_splits WHERE id = ? AND is_settled = ?", splitID, "FALSE").
		Scan(&split.ID, &split.ExpenseID, &split.OwedBy, &split.AmountOwed, &split.CreatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			utils.WriteError(w, "expense split not found", http.StatusNotFound)
			return
		}
		utils.Logger.Errorf("error retrieving expense split: %v", err)
		utils.WriteError(w, "internal server error", http.StatusInternalServerError)
		return
	}

	if split.OwedBy != userID {
		utils.WriteError(w, "this expense split does not belong to you", http.StatusForbidden)
		return
	}

	var payerWallet models.Wallet
	err = db.QueryRowContext(ctx, "SELECT id, balance, last_funded_at, created_at, updated_at FROM wallets WHERE user_id = ?", userID).
		Scan(&payerWallet.ID, &payerWallet.Balance, &payerWallet.LastFundedAt, &payerWallet.CreatedAt, &payerWallet.UpdatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			utils.WriteError(w, "error fetching wallet balance", http.StatusNotFound)
			return
		}
		utils.Logger.Errorf("error fetching wallet balance: %v", err)
		utils.WriteError(w, "internal server error", http.StatusInternalServerError)
		return
	}

	if payerWallet.Balance.LessThan(req.Amount) {
		utils.WriteError(w, "insufficient funds in wallet, please fund wallet", http.StatusPaymentRequired)
		return
	}

	var expense models.GroupExpense
	err = db.QueryRowContext(ctx, "SELECT paid_by FROM group_expenses WHERE id = ?", split.ExpenseID).Scan(&expense.PaidBy)
	if err != nil {
		if err == sql.ErrNoRows {
			utils.WriteError(w, "expense not found", http.StatusNotFound)
			return
		}
		utils.WriteError(w, "internal server error", http.StatusInternalServerError)
		return
	}

	var owedWallet models.Wallet
	err = db.QueryRowContext(ctx, "SELECT id, balance, last_funded_at, created_at, updated_at FROM wallets WHERE user_id = ?", expense.PaidBy).
		Scan(&owedWallet.ID, &owedWallet.Balance, &owedWallet.LastFundedAt, &owedWallet.CreatedAt, &owedWallet.UpdatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			utils.WriteError(w, "error fetching wallet balance", http.StatusNotFound)
			return
		}
		utils.Logger.Errorf("error fetching wallet balance: %v", err)
		utils.WriteError(w, "internal server error", http.StatusInternalServerError)
		return
	}

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		utils.Logger.Error("error starting transaction")
		utils.WriteError(w, "internal server error", http.StatusInternalServerError)
		return
	}

	newPayerBalance := payerWallet.Balance.Sub(req.Amount)
	newOwedBalance := owedWallet.Balance.Add(req.Amount)

	_, err = tx.ExecContext(ctx, `
		UPDATE wallets SET balance = ? WHERE id = ?
	`, newPayerBalance, payerWallet.ID)
	if err != nil {
		tx.Rollback()
		utils.Logger.Errorf("failed to update payer wallet: %v", err)
		utils.WriteError(w, "failed to update payer wallet", http.StatusInternalServerError)
		return
	}

	_, err = tx.ExecContext(ctx, `
		UPDATE wallets SET balance = ?, last_funded_at = ? WHERE id = ?
	`, newOwedBalance, time.Now().Format("2006-01-02 15:04:05"), owedWallet.ID)
	if err != nil {
		tx.Rollback()
		utils.Logger.Errorf("failed to update owed wallet: %v", err)
		utils.WriteError(w, "failed to update owed wallet", http.StatusInternalServerError)
		return
	}

	isFullyPaid := req.Amount.Equal(split.AmountOwed)
	if isFullyPaid {
		_, err = tx.ExecContext(ctx, `
			UPDATE group_expense_splits SET amount_owed = ?, is_settled = TRUE WHERE id = ?
		`, 0, split.ID)
		if err != nil {
			tx.Rollback()
			utils.Logger.Errorf("error marking split as settled: %v", err)
			utils.WriteError(w, "failed to mark split as settled", http.StatusInternalServerError)
			return
		}
	} else {
		remaining := split.AmountOwed.Sub(req.Amount)
		_, err = tx.ExecContext(ctx, `
			UPDATE group_expense_splits SET amount_owed = ? WHERE id = ?
		`, remaining, split.ID)
		if err != nil {
			tx.Rollback()
			utils.Logger.Errorf("failed to update remaining amount: %v", err)
			utils.WriteError(w, "failed to update remaining amount", http.StatusInternalServerError)
			return
		}
	}

	payerRef := fmt.Sprintf("splt-%s", utils.GenerateRandomString(10))
	receiverRef := fmt.Sprintf("splt-%s", utils.GenerateRandomString(10))

	// Payer transaction (DEBIT)
	_, err = tx.ExecContext(ctx, `
		INSERT INTO transactions (user_id, transaction_type, category, amount, status, reference, description)
		VALUES (?, 'debit', 'split', ?, 'success', ?, ?)
	`, userID, req.Amount, payerRef, fmt.Sprintf("Payment for split #%d", split.ID))
	if err != nil {
		tx.Rollback()
		utils.Logger.Errorf("failed to record payer transaction: %v", err)
		utils.WriteError(w, "failed to record payer transaction", http.StatusInternalServerError)
		return
	}

	// Recipient transaction (CREDIT)
	_, err = tx.ExecContext(ctx, `
		INSERT INTO transactions (user_id, transaction_type, category, amount, status, reference, description)
		VALUES (?, 'credit', 'split', ?, 'success', ?, ?)
	`, expense.PaidBy, req.Amount, receiverRef, fmt.Sprintf("Received payment for split #%d", split.ID))
	if err != nil {
		tx.Rollback()
		utils.Logger.Errorf("ailed to record recipient transaction: %v", err)
		utils.WriteError(w, "failed to record recipient transaction", http.StatusInternalServerError)
		return
	}

	if err := tx.Commit(); err != nil {
		utils.Logger.Errorf("transaction commit failed: %v", err)
		utils.WriteError(w, "internal server error", http.StatusInternalServerError)
		return
	}

	var payerName, receiverEmail, groupName string
	db.QueryRowContext(ctx, "SELECT username FROM users WHERE id = ?", userID).Scan(&payerName)
	db.QueryRowContext(ctx, "SELECT email FROM users WHERE id = ?", expense.PaidBy).Scan(&receiverEmail)
	db.QueryRowContext(ctx, `
		SELECT g.name FROM groups g 
		JOIN group_expenses e ON g.id = e.group_id 
		WHERE e.id = ?`, split.ExpenseID).Scan(&groupName)

	go func() {
		if err := utils.SendPaymentReceivedEmail(receiverEmail, payerName, req.Amount.String(), groupName, split.ID, time.Now()); err != nil {
			utils.Logger.Errorf("failed to send payment received email to %s: %v", receiverEmail, err)
		}
	}()

	message := "split payment recorded"
	if isFullyPaid {
		message = "split fully settled"
	}

	utils.WriteJSON(w, map[string]interface{}{
		"status":  "success",
		"message": message,
		"data": map[string]interface{}{
			"amount_paid":      req.Amount,
			"remaining_owed":   split.AmountOwed.Sub(req.Amount),
			"is_fully_settled": isFullyPaid,
		},
	})
}

// FUNC TO DELETE EXPENSE AND RELATED EXPENSE SPLITS
func DeleteExpenseHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		utils.WriteError(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	db := sqlconnect.DB
	if db == nil {
		utils.WriteError(w, "internal server error", http.StatusInternalServerError)
		return
	}

	idStr := r.PathValue("expense_id")
	expenseID, err := strconv.Atoi(idStr)
	if err != nil {
		utils.WriteError(w, "invalid group ID", http.StatusBadRequest)
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

	var expense models.GroupExpense
	err = db.QueryRowContext(ctx, "SELECT id, group_id, paid_by, description, amount FROM group_expenses WHERE id = ?", expenseID).
		Scan(&expense.ID, &expense.GroupID, &expense.PaidBy, &expense.Description, &expense.Amount)
	if err != nil {
		if err == sql.ErrNoRows {
			utils.WriteError(w, "expense not found", http.StatusNotFound)
			return
		}
		utils.WriteError(w, "failed to retrieve expense", http.StatusInternalServerError)
		return
	}

	if expense.PaidBy != userID {
		utils.WriteError(w, "you are not authorized to delete this expense entry", http.StatusUnauthorized)
		return
	}

	res, err := db.ExecContext(ctx, "DELETE FROM group_expenses WHERE id = ?", expenseID)
	if err != nil {
		utils.Logger.Error("unable to delete")
		utils.WriteError(w, "error deleting expense", http.StatusInternalServerError)
		return
	}

	rowsAffected, _ := res.RowsAffected()
	if rowsAffected == 0 {
		utils.Logger.Errorf("error deleting expense: %v", err)
		utils.WriteError(w, "expense not found or already deleted", http.StatusNotFound)
		return
	}

	utils.WriteJSON(w, map[string]interface{}{
		"status":  "success",
		"message": "expense deleted successfully",
	})
}
