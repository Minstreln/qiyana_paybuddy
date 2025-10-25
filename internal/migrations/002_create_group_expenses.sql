CREATE TABLE IF NOT EXISTS group_expenses (
    id INT AUTO_INCREMENT PRIMARY KEY,
    group_id INT NOT NULL,
    paid_by INT NOT NULL,
    description VARCHAR(255) NOT NULL,
    amount DECIMAL(18, 2) NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT fk_expense_group FOREIGN KEY (group_id) REFERENCES groups(id) ON DELETE CASCADE,
    CONSTRAINT fk_expense_user FOREIGN KEY (paid_by) REFERENCES users(id) ON DELETE CASCADE
);