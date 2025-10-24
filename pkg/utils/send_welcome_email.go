package utils

import (
	"fmt"
	"time"
)

func SendWelcomeEmail(to, username string) error {
	subject := fmt.Sprintf("🎉 Welcome to Qiyana Pay Buddy, %s!", username)

	body := fmt.Sprintf(`
	<!DOCTYPE html>
	<html lang="en">
	<head>
		<meta charset="UTF-8" />
		<meta name="viewport" content="width=device-width, initial-scale=1.0" />
		<title>Welcome to Qiyana Pay Buddy</title>

		<!-- Google Font: Poppins -->
		<link href="https://fonts.googleapis.com/css2?family=Poppins:wght@400;500;600;700&display=swap" rel="stylesheet">
		<style>
			body {
				font-family: 'Poppins', sans-serif;
				background-color: #f9fbfa;
				margin: 0;
				padding: 0;
			}
			.container {
				max-width: 650px;
				margin: 40px auto;
				background: #ffffff;
				border-radius: 18px;
				box-shadow: 0 10px 30px rgba(0, 0, 0, 0.08);
				overflow: hidden;
				border-top: 6px solid #00795f;
				position: relative;
			}
			.header {
				background-color: #00795f;
				color: #ffffff;
				text-align: center;
				padding: 40px 20px 20px;
				position: relative;
			}
			.header img {
				width: 80px;
				height: 80px;
				border-radius: 50%%;
				margin-bottom: 15px;
			}
			.header h1 {
				margin: 0;
				font-size: 26px;
				font-weight: 700;
				letter-spacing: 0.3px;
			}
			.content {
				padding: 35px 40px;
				color: #333333;
			}
			.greeting {
				font-size: 18px;
				font-weight: 600;
				margin-bottom: 12px;
			}
			.message {
				font-size: 15.5px;
				line-height: 1.9;
				color: #444444;
				margin-bottom: 16px;
				letter-spacing: 0.2px;
			}
			.highlight {
				color: #00795f;
				font-weight: 600;
			}
			ul {
				padding-left: 22px;
				margin-top: 8px;
				margin-bottom: 16px;
			}
			ul li {
				margin-bottom: 8px;
				font-size: 15px;
				color: #555555;
				line-height: 1.7;
			}
			.cta {
				margin: 35px 0;
				text-align: center;
			}
			.cta a {
				background-color: #00795f;
				color: #ffffff;
				text-decoration: none;
				padding: 14px 35px;
				border-radius: 10px;
				font-weight: 600;
				font-size: 16px;
				letter-spacing: 0.3px;
				transition: all 0.3s ease;
				box-shadow: 0 4px 10px rgba(0, 121, 95, 0.2);
			}
			.cta a:hover {
				background-color: #01936f;
				box-shadow: 0 6px 14px rgba(0, 121, 95, 0.25);
			}
			.footer {
				background: #f0f8f4;
				text-align: center;
				padding: 25px;
				font-size: 13px;
				color: #666666;
				border-top: 1px solid #e5e5e5;
				letter-spacing: 0.3px;
			}
			.brand {
				color: #00795f;
				font-weight: 600;
			}
			.gif {
				display: block;
				margin: 25px auto 0; /* Added margin-top */
				width: 100%%;
				max-width: 600px;
				height: 220px; /* Reduced height for elegance */
				object-fit: cover; /* Keeps proportions nice */
				border-radius: 0;
			}
		</style>
	</head>
	<body>
		<div class="container">
			<div class="header">
				<h1>Welcome to Qiyana Pay Buddy 💸</h1>
			</div>

			<img src="https://media3.giphy.com/media/v1.Y2lkPTc5MGI3NjExNWd2ZmU3MnJ4aDh0dHcyejZ2Mnh2aGx1dGZpZzRtdGt3eTNuNXVwZCZlcD12MV9pbnRlcm5hbF9naWZfYnlfaWQmY3Q9Zw/Ae7SI3LoPYj8Q/giphy.gif" alt="Welcome Animation" class="gif">

			<div class="content">
				<p class="greeting">Hey %s 👋,</p>

				<p class="message">
					We’re <span class="highlight">thrilled</span> to welcome you to <span class="highlight">Qiyana Pay Buddy</span> — your smart companion for managing shared payments and group finances effortlessly.
				</p>

				<p class="message">
					With Qiyana Pay Buddy, you can easily create groups, invite friends, split bills, and track every contribution without stress or confusion. Whether you’re roommates sharing rent, a team managing project funds, or friends splitting trip costs — Qiyana makes it <b>seamless, transparent, and secure</b>.
				</p>

				<p class="message">
					✨ <b>Here’s what you can do with Qiyana Pay Buddy:</b>
				</p>
				<ul>
					<li>💰 Create and manage group wallets effortlessly.</li>
					<li>🧾 Track who has paid and who hasn’t — in real time.</li>
					<li>📤 Send invites to friends via email or link.</li>
					<li>📊 View total expenses and contributions with one tap.</li>
					<li>🔔 Get instant notifications for payments and reminders.</li>
				</ul>

				<p class="message">
					We built Qiyana Pay Buddy because we believe <b>money shouldn’t complicate relationships</b>.  
					It should bring people together — and now, with you on board, we’re one step closer to making that happen.
				</p>

				<div class="cta">
					<a href="https://qiyanapaybuddy.com/login" target="_blank">Continue</a>
				</div>

				<p class="message" style="text-align:center;">
					Need help getting started? Just reply to this email — our friendly support team is always happy to help 💚
				</p>
			</div>

			<div class="footer">
				&copy; %d <span class="brand">Qiyana Pay Buddy</span> — Smart. Simple. Shared Payments.
			</div>
		</div>
	</body>
	</html>
	`, username, time.Now().Year())

	return SendEmail(to, subject, body)
}
