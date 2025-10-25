package utils

import (
	"fmt"
	"time"
)

func SendDebtorReminderEmail(to, firstName string, amount string, groupName string, expenseTitle string, dueDate time.Time) error {
	subject := fmt.Sprintf("💰 Reminder: You Still Owe ₦%s for '%s'", amount, expenseTitle)

	body := fmt.Sprintf(`
	<!DOCTYPE html>
	<html lang="en">
	<head>
	<meta charset="UTF-8">
	<meta name="viewport" content="width=device-width, initial-scale=1.0">
	<title>Payment Reminder</title>
	<style>
		body {
			font-family: 'Segoe UI', Roboto, Arial, sans-serif;
			background-color: #f6f8f7;
			margin: 0;
			padding: 0;
			color: #333;
		}
		.container {
			max-width: 480px;
			margin: 25px auto;
			background: #ffffff;
			border-radius: 12px;
			box-shadow: 0 4px 16px rgba(0, 0, 0, 0.08);
			overflow: hidden;
			border-top: 5px solid #d9534f;
		}
		.header {
			background-color: #d9534f;
			color: #ffffff;
			text-align: center;
			padding: 18px 12px;
		}
		.header h1 {
			margin: 0;
			font-size: 18px;
			font-weight: 600;
		}
		.content {
			padding: 20px 18px;
		}
		.message {
			font-size: 14px;
			line-height: 1.6;
			color: #444;
		}
		.amount-box {
			background: #fff6f6;
			border: 1px solid #f1c1c1;
			border-radius: 8px;
			padding: 12px 14px;
			margin: 16px 0;
			text-align: center;
		}
		.amount-box h3 {
			margin: 0;
			color: #d9534f;
			font-size: 16px;
			font-weight: 700;
		}
		.amount-box p {
			margin: 6px 0 0;
			font-size: 13px;
			color: #555;
		}
		.footer {
			background: #f6f6f6;
			text-align: center;
			padding: 14px;
			font-size: 12px;
			color: #777;
			border-top: 1px solid #e5e5e5;
		}
		.brand {
			color: #0a4d3c;
			font-weight: bold;
		}
	</style>
	</head>

	<body>
		<div class="container">
			<div class="header">
				<h1>Payment Reminder 💬</h1>
			</div>
			<div class="content">
				<p class="message">
					Hi %s,<br><br>
					This is a friendly reminder that you still have an outstanding balance of ₦<b>%s</b> 
					for the shared expense <b>'%s'</b> in your group <b>%s</b>.
				</p>

				<div class="amount-box">
					<h3>₦%s Due</h3>
					<p>Group: %s</p>
					<p>Due Since: %s</p>
				</div>

				<p class="message">
					Please log in to <b>Qiyana Pay Buddy</b> to complete your payment and keep your account in good standing.
				</p>

				<p class="message">
					Thank you for keeping your group finances balanced. 💚
				</p>
			</div>
			<div class="footer">
				&copy; %d <span class="brand">Qiyana Pay Buddy</span> — Smarter Sharing. Stronger Bonds.
			</div>
		</div>
	</body>
	</html>
	`, firstName, amount, expenseTitle, groupName, amount, groupName, dueDate.Format("Jan 2, 2006"), time.Now().Year())

	return SendEmail(to, subject, body)
}
