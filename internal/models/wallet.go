package models

import "github.com/shopspring/decimal"

type Wallet struct {
	ID           int             `json:"id,omitempty" db:"id,omitempty"`
	UserID       int             `json:"user_id,omitempty" db:"user_id,omitempty"`
	Balance      decimal.Decimal `json:"balance,omitempty" db:"balance,omitempty"`
	LastFundedAt string          `json:"last_funded_at,omitempty" db:"last_funded_at,omitempty"`
	CreatedAt    string          `json:"created_at,omitempty" db:"created_at,omitempty"`
	UpdatedAt    string          `json:"updated_at,omitempty" db:"updated_at,omitempty"`
}
