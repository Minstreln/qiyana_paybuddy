package groups

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"qiyana_paybuddy/internal/models"
	"qiyana_paybuddy/internal/repositories/sqlconnect"
	"qiyana_paybuddy/pkg/utils"
	"strconv"
	"strings"
	"time"
)

// FUNC TO CREATE A GROUP
func CreateGroupHandler(w http.ResponseWriter, r *http.Request) {
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

	idFloat, ok := r.Context().Value(utils.ContextKey("userId")).(float64)
	if !ok {
		utils.WriteError(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	userID := int(idFloat)

	var newGroup models.Group
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&newGroup); err != nil {
		utils.WriteError(w, "invalid request body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	newGroup.Name = strings.TrimSpace(newGroup.Name)
	if newGroup.Name == "" || newGroup.Description == "" {
		utils.WriteError(w, "group name and description is required", http.StatusBadRequest)
		return
	}

	if newGroup.Name != "" && strings.TrimSpace(newGroup.Name) == "" {
		utils.WriteError(w, "name cannot be empty or whitespace", http.StatusBadRequest)
		return
	}

	if len(newGroup.Name) > 100 || len(newGroup.Description) > 500 {
		utils.WriteError(w, "name or description too long", http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	tx, err := db.Begin()
	if err != nil {
		utils.Logger.Errorf("failed to start transaction: %v", err)
		utils.WriteError(w, "internal server error", http.StatusInternalServerError)
		return
	}

	query := `INSERT INTO groups (name, description, created_by) VALUES (?, ?, ?)`
	res, err := tx.ExecContext(ctx, query, newGroup.Name, newGroup.Description, userID)
	if err != nil {
		tx.Rollback()
		utils.Logger.Errorf("failed to create group: %v", err)
		utils.WriteError(w, "failed to create group, try again later!", http.StatusInternalServerError)
		return
	}

	id, err := res.LastInsertId()
	if err != nil {
		tx.Rollback()
		utils.Logger.Errorf("failed to get last inserted ID: %v", err)
		utils.WriteError(w, "internal server error", http.StatusInternalServerError)
		return
	}

	insertMemberQuery := `INSERT INTO group_members (group_id, user_id, role) VALUES (?, ?, 'admin')`
	_, err = tx.ExecContext(ctx, insertMemberQuery, id, userID)
	if err != nil {
		tx.Rollback()
		utils.Logger.Errorf("failed to assign group admin: %v", err)
		utils.WriteError(w, "failed to assign group admin", http.StatusInternalServerError)
		return
	}

	if err = tx.Commit(); err != nil {
		tx.Rollback()
		utils.Logger.Errorf("failed to commit transaction: %v", err)
		utils.WriteError(w, "internal server error", http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"status":  "success",
		"message": "Group created successfully",
		"data": map[string]interface{}{
			"group_id":   id,
			"group_name": newGroup.Name,
			"role":       "admin",
		},
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(response)

}

// FUNC TO UPDATE GROUP NAME/DESCRIPTION
func UpdateGroupHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPatch {
		utils.WriteError(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	idStr := r.PathValue("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		utils.WriteError(w, "invalid group ID", http.StatusBadRequest)
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
		Name        string `json:"name"`
		Description string `json:"description"`
	}

	var req request
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err = decoder.Decode(&req); err != nil {
		utils.WriteError(w, "invalid request body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	if req.Name != "" && strings.TrimSpace(req.Name) == "" {
		utils.WriteError(w, "name cannot be empty or whitespace", http.StatusBadRequest)
		return
	}

	if len(req.Name) > 100 || len(req.Description) > 500 {
		utils.WriteError(w, "name or description too long", http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	var createdBy int
	err = db.QueryRowContext(ctx, "SELECT created_by FROM groups WHERE id = ?", id).Scan(&createdBy)
	if err != nil {
		if err == sql.ErrNoRows {
			utils.WriteError(w, "group not found", http.StatusNotFound)
			return
		}
		utils.WriteError(w, "internal server error", http.StatusInternalServerError)
	}

	if createdBy != userID {
		utils.WriteError(w, "forbidden: not group admin", http.StatusForbidden)
		return
	}

	// Build dynamic update query
	fields := []string{}
	args := []interface{}{}

	if req.Name != "" {
		fields = append(fields, "name = ?")
		args = append(args, req.Name)
	}
	if req.Description != "" {
		fields = append(fields, "description = ?")
		args = append(args, req.Description)
	}

	if len(fields) == 0 {
		utils.WriteError(w, "no updates provided", http.StatusBadRequest)
		return
	}

	args = append(args, id)

	query := fmt.Sprintf("UPDATE groups SET %s WHERE id = ?", strings.Join(fields, ", "))
	_, err = db.ExecContext(ctx, query, args...)
	if err != nil {
		utils.WriteError(w, "failed to update group", http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"status":  "success",
		"message": "Group updated successfully",
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// FUNC TO GET ALL GROUPS CREATED BY THE LOGGED-IN ADMIN
func GetMyGroupsHandler(w http.ResponseWriter, r *http.Request) {
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

	query := `
		SELECT id, name, description, created_by, total_expense, created_at
		FROM groups
		WHERE created_by = ?
	`
	args := []interface{}{userID}

	query, args = utils.AddFilters(r, query, args)
	query = utils.AddSorting(r, query)

	rows, err := db.Query(query, args...)
	if err != nil {
		utils.Logger.Errorf("internal server error: %v", err)
		utils.WriteError(w, "internal server error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	groupList := make([]models.Group, 0)
	for rows.Next() {
		var group models.Group
		err := rows.Scan(&group.ID, &group.Name, &group.Description, &group.CreatedBy, &group.TotalExpense, &group.CreatedAt)
		if err != nil {
			utils.Logger.Errorf("error fetching data: %v", err)
			utils.WriteError(w, "internal server error", http.StatusInternalServerError)
			return
		}
		groupList = append(groupList, group)
	}

	response := struct {
		Status string         `json:"status"`
		Count  int            `json:"count"`
		Data   []models.Group `json:"data"`
	}{
		Status: "success",
		Count:  len(groupList),
		Data:   groupList,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// FUNC TO GET A SINGLE GROUP AND ITS MEMBERS
func GetGroupByIDHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		utils.WriteError(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	idStr := r.PathValue("id")
	groupID, err := strconv.Atoi(idStr)
	if err != nil {
		utils.WriteError(w, "invalid group ID", http.StatusBadRequest)
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

	var exists bool
	err = db.QueryRow(`
        SELECT EXISTS(
            SELECT 1 FROM groups g
            LEFT JOIN group_members gm ON gm.group_id = g.id
            WHERE g.id = ? AND (g.created_by = ? OR gm.user_id = ?)
        )
    `, groupID, userID, userID).Scan(&exists)
	if err != nil {
		utils.Logger.Errorf("error checking access: %v", err)
		utils.WriteError(w, "internal server error", http.StatusInternalServerError)
		return
	}
	if !exists {
		utils.WriteError(w, "forbidden: you are not a member of this group", http.StatusForbidden)
		return
	}

	var group models.Group
	err = db.QueryRow(`
        SELECT id, name, description, created_by, total_expense, created_at, updated_at
        FROM groups WHERE id = ?
    `, groupID).Scan(
		&group.ID, &group.Name, &group.Description,
		&group.CreatedBy, &group.TotalExpense,
		&group.CreatedAt, &group.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			utils.WriteError(w, "group not found", http.StatusNotFound)
			return
		}
		utils.Logger.Errorf("error fetching group: %v", err)
		utils.WriteError(w, "internal server error", http.StatusInternalServerError)
		return
	}

	rows, err := db.Query(`
        SELECT id, group_id, user_id, role, joined_at
        FROM group_members
        WHERE group_id = ?
    `, groupID)
	if err != nil {
		utils.Logger.Errorf("error fetching group members: %v", err)
		utils.WriteError(w, "internal server error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	userIDs := make([]interface{}, 0)
	tempMembers := make([]models.GroupMember, 0)
	for rows.Next() {
		var member models.GroupMember
		if err := rows.Scan(&member.ID, &member.GroupID, &member.UserID, &member.Role, &member.JoinedAt); err != nil {
			utils.Logger.Errorf("error scanning group member: %v", err)
			utils.WriteError(w, "internal server error", http.StatusInternalServerError)
			return
		}
		userIDs = append(userIDs, member.UserID)
		tempMembers = append(tempMembers, member)
	}
	if err := rows.Err(); err != nil {
		utils.Logger.Errorf("error iterating group members: %v", err)
		utils.WriteError(w, "internal server error", http.StatusInternalServerError)
		return
	}

	type MemberDetails struct {
		ID        int    `json:"id"`
		GroupID   int    `json:"group_id"`
		UserID    int    `json:"user_id"`
		Role      string `json:"role"`
		JoinedAt  string `json:"joined_at"`
		FirstName string `json:"first_name"`
		LastName  string `json:"last_name"`
		Username  string `json:"username"`
		Email     string `json:"email"`
	}

	members := make([]MemberDetails, 0)
	if len(userIDs) > 0 {
		placeholders := make([]string, len(userIDs))
		for i := range userIDs {
			placeholders[i] = "?"
		}
		query := fmt.Sprintf(`
            SELECT first_name, last_name, username, email, id
            FROM users WHERE id IN (%s)
        `, strings.Join(placeholders, ","))

		userRows, err := db.Query(query, userIDs...)
		if err != nil {
			utils.Logger.Errorf("error fetching user details: %v", err)
			utils.WriteError(w, "internal server error", http.StatusInternalServerError)
			return
		}
		defer userRows.Close()

		userMap := make(map[int]models.User)
		for userRows.Next() {
			var user models.User
			if err := userRows.Scan(&user.FirstName, &user.LastName, &user.Username, &user.Email, &user.ID); err != nil {
				utils.Logger.Errorf("error scanning user details: %v", err)
				utils.WriteError(w, "internal server error", http.StatusInternalServerError)
				return
			}
			userMap[user.ID] = user
		}
		if err := userRows.Err(); err != nil {
			utils.Logger.Errorf("error iterating user details: %v", err)
			utils.WriteError(w, "internal server error", http.StatusInternalServerError)
			return
		}

		for _, member := range tempMembers {
			joinedAt := ""
			if member.JoinedAt.Valid {
				joinedAt = member.JoinedAt.String
			}

			user, exists := userMap[member.UserID]
			if !exists {
				utils.Logger.Warnf("user not found for user_id %d", member.UserID)
				members = append(members, MemberDetails{
					ID:       member.ID,
					GroupID:  member.GroupID,
					UserID:   member.UserID,
					Role:     member.Role,
					JoinedAt: joinedAt,
				})
				continue
			}
			memberDetails := MemberDetails{
				ID:        member.ID,
				GroupID:   member.GroupID,
				UserID:    member.UserID,
				Role:      member.Role,
				JoinedAt:  joinedAt,
				FirstName: user.FirstName,
				LastName:  user.LastName,
				Username:  user.Username,
				Email:     user.Email,
			}
			members = append(members, memberDetails)
		}
	}

	response := struct {
		Status  string          `json:"status"`
		Group   models.Group    `json:"group"`
		Members []MemberDetails `json:"members"`
	}{
		Status:  "success",
		Group:   group,
		Members: members,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// FUNC TO DELETE A GROUP BY ADMIN
func DeleteGroupByHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		utils.WriteError(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	idStr := r.PathValue("id")
	groupID, err := strconv.Atoi(idStr)
	if err != nil {
		utils.WriteError(w, "invalid group ID", http.StatusBadRequest)
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

	var createdBy int
	err = db.QueryRowContext(ctx, "SELECT created_by FROM groups WHERE id = ?", groupID).Scan(&createdBy)
	if err != nil {
		if err == sql.ErrNoRows {
			utils.WriteError(w, "group not found", http.StatusNotFound)
			return
		}
		utils.WriteError(w, "internal server error", http.StatusInternalServerError)
	}

	if createdBy != userID {
		utils.WriteError(w, "forbidden: not group admin", http.StatusForbidden)
		return
	}

	res, err := db.ExecContext(ctx, "DELETE FROM groups WHERE id = ?", groupID)
	if err != nil {
		utils.Logger.Errorf("error deleting data: %v", err)
		utils.WriteError(w, "error deleting group", http.StatusInternalServerError)
		return
	}

	rowsAffected, _ := res.RowsAffected()
	if rowsAffected == 0 {
		utils.Logger.Errorf("error deleting data: %v", err)
		utils.WriteError(w, "group not found or already deleted", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	response := map[string]interface{}{
		"status":  "success",
		"message": "group and its members deleted successfully",
	}

	json.NewEncoder(w).Encode(response)

}

// FUNC TO INVITE MEMBERS TO GROUP
func InviteMembersHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		utils.WriteError(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	idStr := r.PathValue("id")
	groupID, err := strconv.Atoi(idStr)
	if err != nil {
		utils.WriteError(w, "invalid group ID", http.StatusBadRequest)
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

	type InviteRequest struct {
		Email string `json:"email"`
	}

	var invites []InviteRequest
	body, err := io.ReadAll(r.Body)
	if err != nil {
		utils.WriteError(w, "invalid request body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	if err = json.Unmarshal(body, &invites); err != nil {
		utils.WriteError(w, "invalid JSON array", http.StatusBadRequest)
		return
	}

	if len(invites) == 0 {
		utils.WriteError(w, "no invites provided", http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		utils.Logger.Errorf("failed to start transaction: %v", err)
		utils.WriteError(w, "failed to start transaction", http.StatusInternalServerError)
		return
	}

	var group models.Group
	err = tx.QueryRowContext(ctx, "SELECT name, description, created_by FROM groups WHERE id = ?", groupID).
		Scan(&group.Name, &group.Description, &group.CreatedBy)
	if err != nil {
		tx.Rollback()
		utils.WriteError(w, "group not found", http.StatusNotFound)
		return
	}

	if group.CreatedBy != userID {
		tx.Rollback()
		utils.WriteError(w, "forbidden: not group admin", http.StatusForbidden)
		return
	}

	durationDays, err := strconv.Atoi(os.Getenv("INVITE_TOKEN_EXP_DURATION"))
	if err != nil {
		tx.Rollback()
		utils.ErrorHandler(err, "invalid invite token duration")
		return
	}

	expiryTime := time.Now().Add(time.Hour * 24 * time.Duration(durationDays))
	expiry := expiryTime.UTC().Format("2006-01-02 15:04:05")

	stmt, err := tx.PrepareContext(ctx, `
		INSERT INTO group_invitations (group_id, email, token, invited_by, expires_at)
		VALUES (?, ?, ?, ?, ?)
	`)
	if err != nil {
		tx.Rollback()
		utils.ErrorHandler(err, "failed to prepare insert statement")
		return
	}
	defer stmt.Close()

	addedInvites := 0
	skippedInvites := 0
	var successfulInvites []string
	var skippedDetails []map[string]string

	for _, inv := range invites {
		email := strings.TrimSpace(inv.Email)
		if email == "" {
			skippedInvites++
			skippedDetails = append(skippedDetails, map[string]string{
				"email":  email,
				"reason": "email is empty or invalid",
			})
			continue
		}

		var exists bool
		err = tx.QueryRowContext(ctx, `
			SELECT EXISTS(
				SELECT 1 FROM group_invitations WHERE group_id = ? AND email = ?
			)
		`, groupID, email).Scan(&exists)
		if err == nil && exists {
			skippedInvites++
			skippedDetails = append(skippedDetails, map[string]string{
				"email":  email,
				"reason": "user already invited to this group, use the resend invite endpoint",
			})
			continue
		}

		err = tx.QueryRowContext(ctx, `
			SELECT EXISTS(
				SELECT 1 FROM group_members WHERE group_id = ? AND user_id = (
					SELECT id FROM users WHERE email = ?
				)
			)
		`, groupID, email).Scan(&exists)
		if err == nil && exists {
			skippedInvites++
			skippedDetails = append(skippedDetails, map[string]string{
				"email":  email,
				"reason": "user is already a group member",
			})
			continue
		}

		tokenBytes := make([]byte, 32)
		_, err := rand.Read(tokenBytes)
		if err != nil {
			tx.Rollback()
			utils.ErrorHandler(err, "failed to generate token")
			return
		}

		token := hex.EncodeToString(tokenBytes)
		hashedToken := sha256.Sum256(tokenBytes)
		hashedTokenString := hex.EncodeToString(hashedToken[:])

		_, err = stmt.ExecContext(ctx, groupID, email, hashedTokenString, userID, expiry)
		if err != nil {
			tx.Rollback()
			utils.Logger.Errorf("failed to insert invitation for %s: %v", email, err)
			return
		}

		addedInvites++
		successfulInvites = append(successfulInvites, email)

		inviteLink := fmt.Sprintf("https://localhost:3000/groups/invite/%s", token)
		go func(email string, link string) {
			time.AfterFunc(500*time.Millisecond, func() {
				if err := utils.SendGroupInviteEmail(email, group.Name, group.Description, link, expiryTime); err != nil {
					utils.Logger.Errorf("failed to send invite email to %s: %v", email, err)
				}
			})
		}(email, inviteLink)
	}

	if err = tx.Commit(); err != nil {
		tx.Rollback()
		utils.Logger.Errorf("failed to commit transaction: %v", err)
		utils.WriteError(w, "failed to save invites", http.StatusInternalServerError)
		return
	}

	resp := map[string]interface{}{
		"status":            "success",
		"added_invites":     addedInvites,
		"skipped_invites":   skippedInvites,
		"successful_emails": successfulInvites,
		"skipped_details":   skippedDetails,
		"message":           fmt.Sprintf("%d invites sent, %d skipped", addedInvites, skippedInvites),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(resp)
}

// FUNC TO ACCEPT GROUP INVITATION
func AcceptInvitationHandler(w http.ResponseWriter, r *http.Request) {
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

	idFloat, ok := r.Context().Value(utils.ContextKey("userId")).(float64)
	if !ok {
		utils.WriteError(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	userID := int(idFloat)

	token := r.PathValue("tokenCode")

	bytes, err := hex.DecodeString(token)
	if err != nil {
		utils.ErrorHandler(err, "internal error")
		return
	}

	hashedToken := sha256.Sum256(bytes)
	hashedTokenString := hex.EncodeToString(hashedToken[:])

	var id int
	err = db.QueryRow("SELECT id FROM users WHERE id = ?", userID).Scan(&id)
	if err != nil {
		if err == sql.ErrNoRows {
			utils.WriteError(w, "user not found, please sign up", http.StatusNotFound)
			return
		}
		utils.Logger.Errorf("internal server error: %v", err)
		utils.WriteError(w, "internal server error", http.StatusInternalServerError)
		return
	}

	var groupInvite models.GroupInvitation
	query := "SELECT id, group_id, email, status FROM group_invitations WHERE token = ? AND expires_at > ?"
	err = db.QueryRow(query, hashedTokenString, time.Now().Format("2006-01-02 15:04:05")).Scan(&groupInvite.ID, &groupInvite.GroupID, &groupInvite.Email, &groupInvite.Status)
	if err != nil {
		utils.WriteError(w, "invite token expired or invalid", http.StatusBadRequest)
		return
	}

	if groupInvite.Status == "accepted" {
		utils.WriteError(w, "invitation already accepted", http.StatusBadRequest)
		return
	}

	if groupInvite.Status == "expired" {
		utils.WriteError(w, "invitation already expired", http.StatusBadRequest)
		return
	}

	if groupInvite.Status == "revoked" {
		utils.WriteError(w, "invitation revoked by admin", http.StatusBadRequest)
		return
	}

	var exists int
	checkMemberQuery := "SELECT COUNT(*) FROM group_members WHERE group_id = ? AND user_id = ?"
	err = db.QueryRow(checkMemberQuery, groupInvite.GroupID, userID).Scan(&exists)
	if err != nil {
		utils.Logger.Errorf("failed to check existing membership: %v", err)
		utils.WriteError(w, "internal server error", http.StatusInternalServerError)
		return
	}
	if exists > 0 {
		utils.WriteError(w, "you are already a member of this group", http.StatusBadRequest)
		return
	}

	tx, err := db.Begin()
	if err != nil {
		utils.Logger.Errorf("failed to start transaction: %v", err)
		utils.WriteError(w, "internal server error", http.StatusInternalServerError)
		return
	}

	deleteQuery := "DELETE FROM group_invitations WHERE id = ?"
	_, err = tx.Exec(deleteQuery, groupInvite.ID)
	if err != nil {
		tx.Rollback()
		utils.Logger.Errorf("error deleting invite: %v", err)
		utils.WriteError(w, "unable to join group at the moment, please try again later!", http.StatusInternalServerError)
		return
	}

	insertMemberQuery := `INSERT INTO group_members (group_id, user_id, role) VALUES (?, ?, 'member')`
	_, err = tx.Exec(insertMemberQuery, groupInvite.GroupID, userID)
	if err != nil {
		tx.Rollback()
		utils.Logger.Errorf("failed to join group: %v", err)
		utils.WriteError(w, "failed to join group", http.StatusInternalServerError)
		return
	}

	if err = tx.Commit(); err != nil {
		tx.Rollback()
		utils.Logger.Errorf("failed to commit transaction: %v", err)
		utils.WriteError(w, "internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  "success",
		"message": "invite accepted successfully",
	})
}

// FUNC TO REMOVE MEMBER
func RemoveGroupMemberHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPatch {
		utils.WriteError(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	idStr := r.PathValue("id")
	groupID, err := strconv.Atoi(idStr)
	if err != nil {
		utils.WriteError(w, "invalid group ID", http.StatusBadRequest)
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
		ID int `json:"id"`
	}

	var req request
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err = decoder.Decode(&req); err != nil {
		utils.WriteError(w, "invalid request body", http.StatusBadRequest)
		return
	}

	var group models.Group
	err = db.QueryRow("SELECT name, description, created_by FROM groups WHERE id = ?", groupID).
		Scan(&group.Name, &group.Description, &group.CreatedBy)
	if err != nil {
		utils.WriteError(w, "group not found", http.StatusNotFound)
		return
	}

	if group.CreatedBy != userID {
		utils.WriteError(w, "forbidden: not group admin", http.StatusForbidden)
		return
	}

	var memberCheck models.GroupMember
	err = db.QueryRow("SELECT id FROM group_members WHERE group_id = ? AND user_id = ?", groupID, req.ID).Scan(&memberCheck.ID)
	if err != nil {
		if err == sql.ErrNoRows {
			utils.WriteError(w, "user is not a member of this group", http.StatusNotFound)
			return
		}
		utils.WriteError(w, "internal server error", http.StatusInternalServerError)
		return
	}

	if req.ID == userID {
		utils.WriteError(w, "group admins cannot leave. Transfer ownership or delete the group.", http.StatusBadRequest)
		return
	}

	_, err = db.Exec("DELETE FROM group_members WHERE group_id = ? AND user_id = ?", groupID, req.ID)
	if err != nil {
		utils.Logger.Errorf("failed to remove member: %v", err)
		utils.WriteError(w, "failed to remove member", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  "success",
		"message": "member removed successfully",
	})
}

// FUNC FOR A MEMBER TO LEAVE GROUP
func LeaveGroupHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPatch {
		utils.WriteError(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	idStr := r.PathValue("id")
	groupID, err := strconv.Atoi(idStr)
	if err != nil {
		utils.WriteError(w, "invalid group ID", http.StatusBadRequest)
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

	var group models.Group
	err = db.QueryRow("SELECT name, description, created_by FROM groups WHERE id = ?", groupID).
		Scan(&group.Name, &group.Description, &group.CreatedBy)
	if err != nil {
		utils.WriteError(w, "group not found", http.StatusNotFound)
		return
	}

	var memberCheck models.GroupMember
	err = db.QueryRow("SELECT id FROM group_members WHERE group_id = ? AND user_id = ?", groupID, userID).Scan(&memberCheck.ID)
	if err != nil {
		if err == sql.ErrNoRows {
			utils.WriteError(w, "you are not a member of this group", http.StatusNotFound)
			return
		}
		utils.WriteError(w, "internal server error", http.StatusInternalServerError)
		return
	}

	if group.CreatedBy == userID {
		utils.WriteError(w, "group admins cannot leave. Transfer ownership or delete the group.", http.StatusBadRequest)
		return
	}

	_, err = db.Exec("DELETE FROM group_members WHERE group_id = ? AND user_id = ?", groupID, userID)
	if err != nil {
		utils.Logger.Errorf("failed to leave group: %v", err)
		utils.WriteError(w, "failed to leave group", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  "success",
		"message": "you have successfully left the group",
	})
}

// FUNC TO LIST PENDING INVITES FOR ADMIN
func ListPendingInvitesHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		utils.WriteError(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	idStr := r.PathValue("id")
	groupID, err := strconv.Atoi(idStr)
	if err != nil {
		utils.WriteError(w, "invalid group ID", http.StatusBadRequest)
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

	var group models.Group
	err = db.QueryRow("SELECT name, description, created_by FROM groups WHERE id = ?", groupID).
		Scan(&group.Name, &group.Description, &group.CreatedBy)
	if err != nil {
		utils.WriteError(w, "group not found", http.StatusNotFound)
		return
	}

	if group.CreatedBy != userID {
		utils.WriteError(w, "forbidden: not group admin", http.StatusForbidden)
		return
	}

	page, limit := utils.GetPaginationParams(r)
	offset := (page - 1) * limit

	query := `
		SELECT id, group_id, email, status, invited_by, expires_at, created_at
		FROM group_invitations
		WHERE group_id = ? AND status = ?
	`
	args := []interface{}{groupID, "pending"}

	query = utils.AddSorting(r, query)

	query += " LIMIT ? OFFSET ?"
	args = append(args, limit, offset)

	rows, err := db.Query(query, args...)
	if err != nil {
		utils.WriteError(w, "failed to retrieve invitations", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var invites []models.GroupInvitation
	for rows.Next() {
		var invite models.GroupInvitation
		err := rows.Scan(
			&invite.ID,
			&invite.GroupID,
			&invite.Email,
			&invite.Status,
			&invite.InvitedBy,
			&invite.ExpiresAt,
			&invite.CreatedAt,
		)
		if err != nil {
			utils.WriteError(w, "error scanning invitations", http.StatusInternalServerError)
			return
		}
		invites = append(invites, invite)
	}

	if err = rows.Err(); err != nil {
		utils.WriteError(w, "error reading invitations", http.StatusInternalServerError)
		return
	}

	if len(invites) == 0 {
		w.Header().Set("Content-Type", "application/json")
		response := map[string]interface{}{
			"status":  "success",
			"message": "no pending invitations found",
			"data":    []models.GroupInvitation{},
		}
		json.NewEncoder(w).Encode(response)
		return
	}

	response := struct {
		Status   string                   `json:"status"`
		Count    int                      `json:"count"`
		Page     int                      `json:"page"`
		PageSize int                      `json:"page_size"`
		Data     []models.GroupInvitation `json:"data"`
	}{
		Status:   "success",
		Count:    len(invites),
		Page:     page,
		PageSize: limit,
		Data:     invites,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// FUNC TO GET ONE PENDING INVITE BY ADMIN
func GetOnePendingInviteHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		utils.WriteError(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	groupIDStr := r.PathValue("groupId")
	inviteIDStr := r.PathValue("inviteId")

	groupID, err := strconv.Atoi(groupIDStr)
	if err != nil {
		utils.WriteError(w, "invalid group ID", http.StatusBadRequest)
		return
	}

	inviteID, err := strconv.Atoi(inviteIDStr)
	if err != nil {
		utils.WriteError(w, "invalid invite ID", http.StatusBadRequest)
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

	var group models.Group
	err = db.QueryRow("SELECT name, description, created_by FROM groups WHERE id = ?", groupID).
		Scan(&group.Name, &group.Description, &group.CreatedBy)
	if err != nil {
		utils.WriteError(w, "group not found", http.StatusNotFound)
		return
	}

	if group.CreatedBy != userID {
		utils.WriteError(w, "forbidden: not group admin", http.StatusForbidden)
		return
	}

	var invite models.GroupInvitation
	err = db.QueryRow(`
		SELECT id, group_id, email, status, invited_by, expires_at, created_at
		FROM group_invitations
		WHERE id = ? AND group_id = ? AND status = 'pending'
	`, inviteID, groupID).Scan(
		&invite.ID,
		&invite.GroupID,
		&invite.Email,
		&invite.Status,
		&invite.InvitedBy,
		&invite.ExpiresAt,
		&invite.CreatedAt,
	)
	if err == sql.ErrNoRows {
		utils.WriteError(w, "pending invite not found", http.StatusNotFound)
		return
	} else if err != nil {
		utils.WriteError(w, "failed to retrieve invite", http.StatusInternalServerError)
		return
	}

	response := struct {
		Status string                 `json:"status"`
		Data   models.GroupInvitation `json:"data"`
	}{
		Status: "success",
		Data:   invite,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// FUNC TO RESEND INVITATION
func ResendInviteHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		utils.WriteError(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	groupIDStr := r.PathValue("groupId")
	inviteIDStr := r.PathValue("inviteId")

	groupID, err := strconv.Atoi(groupIDStr)
	if err != nil {
		utils.WriteError(w, "invalid group ID", http.StatusBadRequest)
		return
	}

	inviteID, err := strconv.Atoi(inviteIDStr)
	if err != nil {
		utils.WriteError(w, "invalid invite ID", http.StatusBadRequest)
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

	var group models.Group
	err = db.QueryRow("SELECT name, description, created_by FROM groups WHERE id = ?", groupID).
		Scan(&group.Name, &group.Description, &group.CreatedBy)
	if err != nil {
		utils.WriteError(w, "group not found", http.StatusNotFound)
		return
	}

	if group.CreatedBy != userID {
		utils.WriteError(w, "forbidden: not group admin", http.StatusForbidden)
		return
	}

	var invite models.GroupInvitation
	err = db.QueryRow(`SELECT id, group_id, email, status FROM group_invitations WHERE id = ? AND group_id = ?`, inviteID, groupID).Scan(&invite.ID, &invite.GroupID, &invite.Email, &invite.Status)
	if err == sql.ErrNoRows {
		utils.WriteError(w, "invitation not found", http.StatusNotFound)
		return
	} else if err != nil {
		utils.WriteError(w, "cannot resend a non-pending invitation", http.StatusInternalServerError)
		return
	}

	if invite.Status != "pending" {
		utils.WriteError(w, "cannot resend a non-pending invitation", http.StatusBadRequest)
		return
	}

	durationDays, err := strconv.Atoi(os.Getenv("INVITE_TOKEN_EXP_DURATION"))
	if err != nil {
		utils.ErrorHandler(err, "invalid invite token duration")
		return
	}

	expiryTime := time.Now().Add(time.Hour * 24 * time.Duration(durationDays))
	expiry := expiryTime.UTC().Format("2006-01-02 15:04:05")

	tokenBytes := make([]byte, 32)
	_, err = rand.Read(tokenBytes)
	if err != nil {
		utils.ErrorHandler(err, "failed to generate token")
		return
	}

	token := hex.EncodeToString(tokenBytes)
	hashedToken := sha256.Sum256(tokenBytes)
	hashedTokenString := hex.EncodeToString(hashedToken[:])

	_, err = db.Exec(`
		UPDATE group_invitations 
		SET token = ?, created_at = NOW(), expires_at = ? 
		WHERE id = ? AND group_id = ?`,
		hashedTokenString, expiry, inviteID, groupID)
	if err != nil {
		utils.WriteError(w, "failed to update invitation", http.StatusInternalServerError)
		return
	}

	inviteLink := fmt.Sprintf("https://localhost:3000/groups/invite/%s", token)
	go func(email string, link string) {
		time.AfterFunc(500*time.Millisecond, func() {
			if err := utils.SendGroupInviteEmail(email, group.Name, group.Description, link, expiryTime); err != nil {
				utils.Logger.Errorf("failed to send invite email to %s: %v", email, err)
			}
		})
	}(invite.Email, inviteLink)

	response := map[string]interface{}{
		"status":  "success",
		"message": "invitation resent successfully",
		"data": map[string]interface{}{
			"invite_id":  inviteID,
			"group_id":   groupID,
			"email":      invite.Email,
			"expires_at": expiryTime,
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// FUNC TO REVOKE INVITATION
func RevokeInvitationHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		utils.WriteError(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	idStr := r.PathValue("id")
	inviteID, err := strconv.Atoi(idStr)
	if err != nil {
		utils.WriteError(w, "invalid invitation ID", http.StatusBadRequest)
		return
	}

	db := sqlconnect.DB
	if db == nil {
		utils.Logger.Error("DB not initialized")
		utils.WriteError(w, "internal server error", http.StatusInternalServerError)
		return
	}

	var exists bool
	err = db.QueryRow("SELECT EXISTS(SELECT 1 FROM group_invitations WHERE id = ?)", inviteID).Scan(&exists)
	if err != nil {
		utils.WriteError(w, "failed to check invitation", http.StatusInternalServerError)
		return
	}
	if !exists {
		utils.WriteError(w, "invitation not found", http.StatusNotFound)
		return
	}

	_, err = db.Exec("DELETE FROM group_invitations WHERE id = ?", inviteID)
	if err != nil {
		utils.WriteError(w, "failed to revoke invitation", http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"status":  "success",
		"message": "invitation revoked successfully",
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}
