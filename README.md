# 🪙 QIYANA PAY Buddy

A shared group payment and expense management system built with **Go (Golang)** and **MariaDB**.

**QIYANA PAY Buddy** helps users create groups, add members, record shared expenses, split bills automatically, and settle debts through wallet funding and in-app transactions — built with a **double-entry ledger system** for accuracy and transparency.

---

## 🚀 Features

### 👥 Group Management

- Create and manage groups
- Add and remove members
- Invite members via unique invitation links or codes
- Accept or reject invitations
- Leave a group with proper validation

### 💸 Expense Management

- Add shared expenses within groups
- Split expenses equally among members
- Automatically record payables and receivables
- Update and delete expenses safely using transactions

### 💰 Wallet & Transactions

- Each user has an in-app wallet
- Fund wallet to settle group debts
- Send and receive funds between members
- **Double-entry ledger system:** every debit has a corresponding credit
- **Atomic transactions:** no partial updates, no broken balances

### 🔒 Security & Data Integrity

- Role-based validation for group admins and members
- All critical operations wrapped in database transactions
- Error-safe rollback mechanism
- JWT-based authentication and authorization

### 🧾 Notifications & History

- Track group expense history
- View all wallet transactions
- Detailed logs for debits, credits, and balances

---

## 🧠 Technical Highlights

| Component          | Description                                                                                  |
| ------------------ | -------------------------------------------------------------------------------------------- |
| **Language**       | Go (Golang)                                                                                  |
| **Database**       | MariaDB                                                                                      |
| **Architecture**   | RESTful API with modular handlers                                                            |
| **Transactions**   | Implemented using `db.BeginTx()` for atomicity                                               |
| **Ledger System**  | Double-entry bookkeeping (every debit has a credit)                                          |
| **Error Handling** | Centralized error management with `utils.WriteError`                                         |
| **Logging**        | Structured logging for better traceability                                                   |
| **Routing**        | Organized modular routes (`/groups`, `/group-expense`, `/wallet`, `/users`, `/transactions`) |

---

## API Endpoints

All endpoints are prefixed with `/api/v1/`.

| Category    | Method | Endpoint              | Description                |
| ----------- | ------ | --------------------- | -------------------------- |
| **User**    | POST   | /users/signup         | Register a new user        |
| **Group**   | POST   | /groups/create        | Create a new group         |
| **Expense** | POST   | /group-expense/create | Create a new group expense |
| **Wallet**  | POST   | /wallet/fund          | Fund wallet via Paystack   |

_(See full documentation for additional endpoints.)_

For a complete list of endpoints, refer to the [full API documentation](https://www.postman.com/subsum/workspace/qiyana-pay-buddy/collection/27481035-95a3be19-490f-41d7-9306-47523513a7bc?action=share&creator=27481035&active-environment=27481035-6df19659-b8cc-4884-a786-3941fb0771b1).

### **Base API Prefix**

All endpoints are prefixed with:

/api/v1/

For example:

- `POST /api/v1/users/signup`
- `GET /api/v1/groups/`

---

## ⚙️ Installation & Setup

### 1️⃣ Clone the Repository

```bash
git clone https://github.com/Minstreln/qiyana_paybuddy.git
cd qiyana_paybuddy

go mod tidy

```

For a complete list of endpoints, refer to the [full API documentation](https://www.postman.com/subsum/workspace/qiyana-pay-buddy/collection/27481035-95a3be19-490f-41d7-9306-47523513a7bc?action=share&creator=27481035&active-environment=27481035-6df19659-b8cc-4884-a786-3941fb0771b1).

## ⚙️ Environment variables

```bash

SERVER_PORT=:3000
APP_ENV=development

#### DATABASE CREDENTIALS
DB_USER=root
DB_PASSWORD=<your_db_password>
DB_NAME=qiyana_paybuddy
DB_PORT=3306
HOST=127.0.0.1

#### JWT CREDENTIALS
JWT_SECRET=<your_jwt_secret>
JWT_EXPIRES_IN=6000s

RESET_TOKEN_EXP_DURATION=10
OTP_TOKEN_EXP_DURATION=10
INVITE_TOKEN_EXP_DURATION=3

#### SELF SIGNED CERTS
CERT_FILE="cert.pem"
KEY_FILE="key.pem"

#### PAYSTACK CREDENTIALS

PAYSTACK_SECRET_KEY=<your_paystack_secret_key>
PAYSTACK_PUBLIC_KEY=<your_paystack_public_key>

#### EMAIL CREDENTIALS

SMTP_EMAIL=<your_smtp_email>
SMTP_PASS=<your_smtp_password>
SMTP_HOST=<smtp_host>
SMTP_PORT=465

```

### use the below code to generate cert and key files - you can find the openssl.cnf in the root folder

```bash
# openssl req -x509 -nodes -days 365 -newkey rsa:2048 -keyout key.pem -out cert.pem -config openssl.cnf

```

### CREATE DATABASE

```bash

CREATE DATABASE qiyana_paybuddy;

```

### MIGRATE TABLES

```bash
go to the folder internal/migrations and migrate the tables to your mysql database

```

### START SERVER

```bash

go run cmd/api/server.go

```

For a complete list of endpoints, refer to the [full API documentation](https://www.postman.com/subsum/workspace/qiyana-pay-buddy/collection/27481035-95a3be19-490f-41d7-9306-47523513a7bc?action=share&creator=27481035&active-environment=27481035-6df19659-b8cc-4884-a786-3941fb0771b1).
