package models

import (
	"database/sql"

	"github.com/shopspring/decimal"
)

type GroupExpenseSplit struct {
	ID         int             `json:"id,omitempty" db:"id,omitempty"`
	ExpenseID  int             `json:"expense_id,omitempty" db:"expense_id,omitempty"`
	OwedBy     int             `json:"owed_by,omitempty" db:"owed_by,omitempty"`
	AmountOwed decimal.Decimal `json:"amount_owed,omitempty" db:"amount_owed,omitempty"`
	IsSettled  bool            `json:"is_settled,omitempty" db:"is_settled,omitempty"`
	CreatedAt  sql.NullString  `json:"created_at,omitempty" db:"created_at,omitempty"`
}
