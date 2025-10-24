package models

import "database/sql"

type GroupInvitation struct {
	ID        int            `json:"id,omitempty" db:"id,omitempty"`
	GroupID   int            `json:"group_id,omitempty" db:"group_id,omitempty"`
	Email     string         `json:"email,omitempty" db:"email,omitempty"`
	Token     string         `json:"token,omitempty" db:"token,omitempty"`
	Status    string         `json:"status,omitempty" db:"status,omitempty"`
	InvitedBy int            `json:"invited_by,omitempty" db:"invited_by,omitempty"`
	ExpiresAt sql.NullString `json:"expires_at,omitempty" db:"expires_at,omitempty"`
	CreatedAt sql.NullString `json:"created_at,omitempty" db:"created_at,omitempty"`
}
