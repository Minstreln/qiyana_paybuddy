package models

import (
	"database/sql"

	"github.com/shopspring/decimal"
)

type Group struct {
	ID           int             `json:"id,omitempty" db:"id,omitempty"`
	Name         string          `json:"name,omitempty" db:"name,omitempty"`
	Description  string          `json:"description,omitempty" db:"description,omitempty"`
	CreatedBy    int             `json:"created_by,omitempty" db:"created_by,omitempty"`
	TotalExpense decimal.Decimal `json:"total_expense,omitempty" db:"total_expense,omitempty"`
	CreatedAt    sql.NullString  `json:"created_at,omitempty" db:"created_at,omitempty"`
	UpdatedAt    sql.NullString  `json:"updated_at,omitempty" db:"updated_at,omitempty"`
}
