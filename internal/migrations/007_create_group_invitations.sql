CREATE TABLE IF NOT EXISTS group_invitations (
    id INT AUTO_INCREMENT PRIMARY KEY,
    group_id INT NOT NULL,
    email VARCHAR(100) NOT NULL UNIQUE,
    token VARCHAR(255) UNIQUE,
    status ENUM('pending', 'accepted', 'expired', 'revoked') DEFAULT 'pending',
    invited_by INT NOT NULL,
    expires_at TIMESTAMP NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (group_id) REFERENCES groups(id) ON DELETE CASCADE,
    FOREIGN KEY (invited_by) REFERENCES users(id) ON DELETE CASCADE
);