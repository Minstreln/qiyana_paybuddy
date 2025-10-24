package sqlconnect

import (
	"database/sql"
	"fmt"
	"os"

	_ "github.com/go-sql-driver/mysql"
)

var DB *sql.DB

func ConnectDb() error {
	if DB != nil {
		return nil
	}

	fmt.Println("Connecting to MariaDB...")

	user := os.Getenv("DB_USER")
	password := os.Getenv("DB_PASSWORD")
	dbname := os.Getenv("DB_NAME")
	port := os.Getenv("DB_PORT")
	host := os.Getenv("DB_HOST")

	connectionString := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s", user, password, host, port, dbname)

	var err error
	DB, err = sql.Open("mysql", connectionString)
	if err != nil {
		return fmt.Errorf("failed to open DB connection: %w", err)
	}

	if err = DB.Ping(); err != nil {
		return fmt.Errorf("failed to ping DB: %w", err)
	}

	fmt.Println("✅ Connected to MariaDB")
	return nil
}
