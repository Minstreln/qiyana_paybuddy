CREATE TABLE IF NOT EXISTS users (
	id INT AUTO_INCREMENT PRIMARY KEY,
    first_name VARCHAR(255) NOT NULL,
    last_name VARCHAR(255) NOT NULL,
    email VARCHAR(255) NOT NULL UNIQUE,
    username VARCHAR(255) NOT NULL UNIQUE,
    password VARCHAR(255) NOT NULL,
    password_changed_at VARCHAR(255),
    user_created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    password_reset_token VARCHAR(255),
    password_token_expires VARCHAR(255) DEFAULT NULL,
    otp VARCHAR(255) DEFAULT NULL;
    otp_expires VARCHAR(255) DEFAULT NULL;
    email_confirmed VARCHAR(255) DEFAULT NULL;
    inactive_status BOOLEAN NOT NULL,
    role VARCHAR(50) NOT NULL,
    INDEX idx_email (email),
    INDEX idx_username (username)
)