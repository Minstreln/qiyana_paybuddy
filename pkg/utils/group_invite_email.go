package utils

import (
	"fmt"
	"time"
)

func SendGroupInviteEmail(to, groupName, description, inviteURL string, expiresAt time.Time) error {
	subject := fmt.Sprintf("🌟 You're Invited to Join '%s' on Qiyana Pay Buddy!", groupName)

	body := fmt.Sprintf(`
	<!DOCTYPE html>
	<html lang="en">
	<head>
	<meta charset="UTF-8" />
	<meta name="viewport" content="width=device-width, initial-scale=1.0" />
	<title>Group Invitation</title>
	<style>
		body {
			font-family: 'Segoe UI', Roboto, Arial, sans-serif;
			background-color: #f5f7f6;
			margin: 0;
			padding: 0;
			color: #333333;
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
		.subheader {
			font-size: 13px;
			margin-top: 4px;
			color: #cce8df;
		}
		.content {
			padding: 20px 18px;
		}
		.greeting {
			font-size: 14px;
			font-weight: 500;
			margin-bottom: 10px;
			color: #111111;
		}
		.message {
			font-size: 13px;
			line-height: 1.5;
			color: #444444;
			margin-bottom: 14px;
		}
		.group-box {
			background: #f8fdfa;
			border: 1px solid #d7ece4;
			border-radius: 8px;
			padding: 12px 14px;
			margin: 16px 0;
		}
		.group-box h3 {
			margin: 0;
			color: #0a4d3c;
			font-size: 15px;
		}
		.group-box p {
			margin-top: 4px;
			font-size: 12px;
			color: #555555;
		}
		.btn {
			display: inline-block;
			background-color: #0a4d3c;
			color: #ffffff !important;
			text-decoration: none;
			font-size: 14px;
			font-weight: 600;
			padding: 10px 22px;
			border-radius: 6px;
			margin: 18px 0;
			text-align: center;
			transition: background 0.2s ease;
		}
		.btn:hover {
			background-color: #063428;
		}
		.benefits {
			background: #f1f8f4;
			padding: 14px;
			border-radius: 8px;
			margin-top: 14px;
		}
		.benefits h4 {
			margin: 0 0 8px 0;
			font-size: 14px;
			color: #0a4d3c;
		}
		.benefits ul {
			margin: 0;
			padding-left: 18px;
			color: #444;
		}
		.benefits ul li {
			font-size: 12px;
			margin-bottom: 4px;
		}
		.expiry {
			margin-top: 16px;
			font-size: 12px;
			color: #888888;
		}
		.footer {
			background: #f0f6f2;
			text-align: center;
			padding: 14px;
			font-size: 12px;
			color: #777777;
			border-top: 1px solid #e5e5e5;
		}
		.brand {
			color: #0a4d3c;
			font-weight: bold;
		}

		@media (max-width: 480px) {
			.container {
				width: 92%%;
				margin: 12px auto;
			}
			.content {
				padding: 16px 14px;
			}
			.header h1 {
				font-size: 17px;
			}
			.btn {
				display: block;
				width: 100%%;
				padding: 12px 0;
			}
		}
	</style>
	</head>

	<body>
		<div class="container">
			<div class="header">
				<h1>You're Invited!</h1>
				<p class="subheader">Join your team on Qiyana Pay Buddy</p>
			</div>

			<div class="content">
				<p class="greeting">Hello there,</p>
				<p class="message">
					You’ve been invited to join the group <b>%s</b> on <b>Qiyana Pay Buddy</b> — a simple, smart way for teams and friends to manage shared expenses, stay transparent, and stay connected.
				</p>

				<div class="group-box">
					<h3>%s</h3>
					<p>%s</p>
				</div>

				<div style="text-align: center;">
					<a href="%s" class="btn">Join Group Now</a>
				</div>

				<div class="benefits">
					<h4>Why Qiyana Pay Buddy?</h4>
					<ul>
						<li>💰 Easily split and track shared expenses.</li>
						<li>🤝 Manage members and contributions in real-time.</li>
						<li>📊 Get automatic expense summaries and balances.</li>
						<li>🔒 Secure and private access for every member.</li>
					</ul>
				</div>

				<p class="expiry">
					This invitation link expires on <b>%s</b>.
				</p>
			</div>

			<div class="footer">
				&copy; %d <span class="brand">Qiyana Pay Buddy</span> — Smarter Sharing. Stronger Bonds.
			</div>
		</div>
	</body>
	</html>
	`, groupName, groupName, description, inviteURL, expiresAt.Format("3:04 PM, Jan 2 2006"), time.Now().Year())

	return SendEmail(to, subject, body)
}
