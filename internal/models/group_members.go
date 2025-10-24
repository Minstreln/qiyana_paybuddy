package models

import "database/sql"

type GroupMember struct {
	ID       int            `json:"id,omitempty" db:"id,omitempty"`
	GroupID  int            `json:"group_id,omitempty" db:"group_id,omitempty"`
	UserID   int            `json:"user_id,omitempty" db:"user_id,omitempty"`
	Role     string         `json:"role,omitempty" db:"role,omitempty"`
	JoinedAt sql.NullString `json:"joined_at,omitempty" db:"joined_at,omitempty"`
}
