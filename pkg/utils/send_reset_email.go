package utils

import (
	"fmt"
	"time"
)

// SendPasswordResetEmail sends a password reset email with a secure link.
func SendPasswordResetEmail(to, username, resetURL string, expiresAt time.Time) error {
	subject := "🔐 Reset Your Qiyana Pay Buddy Password"

	body := fmt.Sprintf(`
	<!DOCTYPE html>
	<html lang="en">
	<head>
		<meta charset="UTF-8" />
		<meta name="viewport" content="width=device-width, initial-scale=1.0" />
		<title>Password Reset</title>
		<style>
			body {
				font-family: 'Segoe UI', Roboto, Arial, sans-serif;
				background-color: #f4f8f5;
				margin: 0;
				padding: 0;
			}
			.container {
				max-width: 520px;
				margin: 40px auto;
				background: #ffffff;
				border-radius: 12px;
				box-shadow: 0 8px 24px rgba(0, 0, 0, 0.08);
				overflow: hidden;
				border-top: 5px solid #0a4d3c;
			}
			.header {
				background-color: #0a4d3c;
				color: #ffffff;
				text-align: center;
				padding: 24px 20px;
			}
			.header h1 {
				margin: 0;
				font-size: 22px;
				font-weight: 600;
				letter-spacing: 0.5px;
			}
			.content {
				padding: 30px 35px;
				color: #333333;
			}
			.greeting {
				font-size: 16px;
				font-weight: 500;
				margin-bottom: 12px;
			}
			.message {
				font-size: 15px;
				line-height: 1.6;
				color: #555555;
			}
			.reset-btn {
				display: inline-block;
				background-color: #0a4d3c;
				color: #ffffff !important;
				text-decoration: none;
				font-size: 16px;
				font-weight: 600;
				padding: 12px 22px;
				border-radius: 8px;
				margin: 25px 0;
				text-align: center;
			}
			.expiry {
				margin-top: 18px;
				font-size: 14px;
				color: #888888;
			}
			.footer {
				background: #f0f6f2;
				text-align: center;
				padding: 18px;
				font-size: 13px;
				color: #777777;
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
				<h1>Reset Your Password</h1>
			</div>
			<div class="content">
				<p class="greeting">Hello %s,</p>
				<p class="message">
					We received a request to reset your password. You can reset it by clicking the button below:
				</p>

				<div style="text-align: center;">
					<a href="%s" class="reset-btn">Reset Password</a>
				</div>

				<p class="message">
					This link will expire at <b>%s</b>. If you did not request a password reset, please ignore this email.
				</p>
			</div>
			<div class="footer">
				&copy; %d <span class="brand">Qiyana Pay Buddy</span> — Secure. Fast. Reliable.
			</div>
		</div>
	</body>
	</html>
	`, username, resetURL, expiresAt.Format("3:04 PM, Jan 2 2006"), time.Now().Year())

	return SendEmail(to, subject, body)
}
