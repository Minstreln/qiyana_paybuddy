package utils

import (
	"fmt"
	"time"
)

func SendPaymentReceivedEmail(to, payerName string, amount string, groupName string, splitID int, date time.Time) error {
	subject := fmt.Sprintf("💸 You've Been Paid — Split #%d Settled!", splitID)

	body := fmt.Sprintf(`
	<!DOCTYPE html>
	<html lang="en">
	<head>
	<meta charset="UTF-8">
	<meta name="viewport" content="width=device-width, initial-scale=1.0">
	<title>Payment Received</title>
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
			border-top: 5px solid #0a4d3c;
		}
		.header {
			background-color: #0a4d3c;
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
			background: #f2fdf6;
			border: 1px solid #bfe7cb;
			border-radius: 8px;
			padding: 12px 14px;
			margin: 16px 0;
			text-align: center;
		}
		.amount-box h3 {
			margin: 0;
			color: #0a4d3c;
			font-size: 16px;
			font-weight: 700;
		}
		.amount-box p {
			margin: 6px 0 0;
			font-size: 13px;
			color: #555;
		}
		.footer {
			background: #f0f6f2;
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
				<h1>You've Been Paid 🎉</h1>
			</div>
			<div class="content">
				<p class="message">
					Hi there,<br><br>
					Good news! <b>%s</b> has just sent you a payment of ₦<b>%s</b> for your shared expense in the group <b>%s</b>.
				</p>

				<div class="amount-box">
					<h3>₦%s Received</h3>
					<p>Split ID: #%d</p>
					<p>Date: %s</p>
				</div>

				<p class="message">
					You can view this transaction in your wallet history on <b>Qiyana Pay Buddy</b>.
				</p>
			</div>
			<div class="footer">
				&copy; %d <span class="brand">Qiyana Pay Buddy</span> — Smarter Sharing. Stronger Bonds.
			</div>
		</div>
	</body>
	</html>
	`, payerName, amount, groupName, amount, splitID, date.Format("3:04 PM, Jan 2 2006"), time.Now().Year())

	return SendEmail(to, subject, body)
}
