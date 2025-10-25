package cron

import (
	"context"
	"database/sql"
	"fmt"
	"qiyana_paybuddy/pkg/utils"
	"sync"
	"time"

	"github.com/robfig/cron/v3"
)

func StartCronJob(db *sql.DB) *cron.Cron {
	c := cron.New()

	// Runs every 6 hours — check expired invitations
	_, err := c.AddFunc("0 */6 * * *", func() {
		err := CheckAndUpdateExpiredInvitations(db)
		if err != nil {
			utils.Logger.Errorf("Cron job failed to update expired invitations: %v", err)
		}
	})
	if err != nil {
		utils.Logger.Errorf("Failed to schedule invitation expiration job: %v", err)
	}

	// Runs daily at midnight — send reminders
	_, err = c.AddFunc("0 0 * * *", func() {
		err := SendReminderEmailsToDebtors(db)
		if err != nil {
			utils.Logger.Errorf("Cron job failed to send reminder emails: %v", err)
		}
	})
	if err != nil {
		utils.Logger.Errorf("Failed to schedule debtor reminder job: %v", err)
	}

	c.Start()
	utils.Logger.Info("Cron jobs started (invitation expiry every 6h, debtor reminders daily at midnight)")
	return c
}

// -------------------------------------------------------------
// Check and update expired group invitations
// -------------------------------------------------------------
func CheckAndUpdateExpiredInvitations(db *sql.DB) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}

	result, err := tx.ExecContext(ctx, `
		UPDATE group_invitations 
		SET status = 'expired' 
		WHERE expires_at < ? AND status != 'expired'
	`, time.Now().UTC().Format("2006-01-02 15:04:05"))
	if err != nil {
		tx.Rollback()
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		tx.Rollback()
		return err
	}

	if err := tx.Commit(); err != nil {
		return err
	}

	if rowsAffected > 0 {
		utils.Logger.Infof("Updated %d expired invitations to status 'expired'", rowsAffected)
	}
	return nil
}

// -------------------------------------------------------------
// Send daily reminders to debtors (now runs email sends concurrently)
// -------------------------------------------------------------
func SendReminderEmailsToDebtors(db *sql.DB) error {
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()

	rows, err := db.QueryContext(ctx, `
		SELECT 
			s.owed_by,
			u.email,
			u.first_name,
			g.name AS group_name,
			e.description AS expense_title,
			e.created_at,
			SUM(s.amount_owed) AS total_owed
		FROM group_expense_splits s
		JOIN group_expenses e ON s.expense_id = e.id
		JOIN groups g ON e.group_id = g.id
		JOIN users u ON s.owed_by = u.id
		WHERE s.is_settled = FALSE
		GROUP BY s.owed_by, e.id
	`)
	if err != nil {
		return err
	}
	defer rows.Close()

	var wg sync.WaitGroup
	errChan := make(chan error, 10)

	for rows.Next() {
		var (
			email, firstName, groupName, expenseTitle string
			expenseCreatedAtRaw                       sql.NullString
			totalOwed                                 float64
		)

		if err := rows.Scan(
			new(int),
			&email,
			&firstName,
			&groupName,
			&expenseTitle,
			&expenseCreatedAtRaw,
			&totalOwed,
		); err != nil {
			utils.Logger.Errorf("Failed to scan debtor row: %v", err)
			continue
		}

		var expenseCreatedAt time.Time
		if expenseCreatedAtRaw.Valid {
			expenseCreatedAt, err = time.Parse("2006-01-02 15:04:05", expenseCreatedAtRaw.String)
			if err != nil {
				utils.Logger.Errorf("Failed to parse created_at for %s: %v", email, err)
				continue
			}
		} else {
			expenseCreatedAt = time.Now()
		}

		wg.Add(1)
		go func(email, firstName, groupName, expenseTitle string, totalOwed float64, expenseCreatedAt time.Time) {
			defer wg.Done()

			totalOwedStr := fmt.Sprintf("%.2f", totalOwed)

			if err := utils.SendDebtorReminderEmail(
				email,
				firstName,
				totalOwedStr,
				groupName,
				expenseTitle,
				expenseCreatedAt,
			); err != nil {
				errChan <- fmt.Errorf("failed to send reminder email to %s: %v", email, err)
				return
			}

			utils.Logger.Infof("📧 Sent reminder to %s (%s) — ₦%.2f for '%s' in '%s'",
				firstName, email, totalOwed, expenseTitle, groupName)
		}(email, firstName, groupName, expenseTitle, totalOwed, expenseCreatedAt)
	}

	wg.Wait()
	close(errChan)

	for e := range errChan {
		utils.Logger.Error(e)
	}

	if err := rows.Err(); err != nil {
		utils.Logger.Errorf("Error iterating debtor rows: %v", err)
		return err
	}

	utils.Logger.Info("✅ Finished sending all debtor reminder emails.")
	return nil
}
