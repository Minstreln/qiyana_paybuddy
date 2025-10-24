package models

import (
	"database/sql"

	"github.com/shopspring/decimal"
)

type Transaction struct {
	ID              int             `json:"id,omitempty" db:"id,omitempty"`
	UserID          int             `json:"user_id,omitempty" db:"user_id,omitempty"`
	TransactionType string          `json:"transaction_type,omitempty" db:"transaction_type,omitempty"`
	Category        string          `json:"category,omitempty" db:"category,omitempty"`
	Amount          decimal.Decimal `json:"amount,omitempty" db:"amount,omitempty"`
	Status          string          `json:"status,omitempty" db:"status,omitempty"`
	Reference       string          `json:"reference,omitempty" db:"reference,omitempty"`
	Description     string          `json:"description,omitempty" db:"description,omitempty"`
	CreatedAt       sql.NullString  `json:"created_at,omitempty" db:"created_at,omitempty"`
	UpdatedAt       sql.NullString  `json:"updated_at,omitempty" db:"updated_at,omitempty"`
}
