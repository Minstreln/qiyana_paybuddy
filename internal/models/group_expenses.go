package models

import (
	"database/sql"

	"github.com/shopspring/decimal"
)

type GroupExpense struct {
	ID          int             `json:"id,omitempty" db:"id,omitempty"`
	GroupID     int             `json:"group_id,omitempty" db:"group_id,omitempty"`
	PaidBy      int             `json:"paid_by,omitempty" db:"paid_by,omitempty"`
	Description string          `json:"description,omitempty" db:"description,omitempty"`
	Amount      decimal.Decimal `json:"amount,omitempty" db:"amount,omitempty"`
	CreatedAt   sql.NullString  `json:"created_at,omitempty" db:"created_at,omitempty"`
}
