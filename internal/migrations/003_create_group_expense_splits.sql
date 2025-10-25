CREATE TABLE IF NOT EXISTS group_expense_splits (
    id INT AUTO_INCREMENT PRIMARY KEY,
    expense_id INT NOT NULL,
    owed_by INT NOT NULL,
    amount_owed DECIMAL(18, 2) NOT NULL,
    is_settled BOOLEAN DEFAULT FALSE,
    CONSTRAINT fk_split_expense FOREIGN KEY (expense_id) REFERENCES group_expenses(id) ON DELETE CASCADE,
    CONSTRAINT fk_split_user FOREIGN KEY (owed_by) REFERENCES users(id) ON DELETE CASCADE
);