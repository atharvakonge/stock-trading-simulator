package db

import (
	"database/sql"
	"fmt"
	"log"
	"testing"
	"time"
)

// SetupTestDB creates a test database connection
func SetupTestDB(t *testing.T) *sql.DB {
	connStr := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		"localhost",
		"5433",
		"trader",
		"trading123",
		"trading_db",
	)

	db, err := sql.Open("postgres", connStr)
	if err != nil {
		t.Fatalf("Failed to connect to test database: %v", err)
	}

	if err = db.Ping(); err != nil {
		t.Fatalf("Failed to ping test database: %v", err)
	}

	// Set global DB for handlers
	DB = db

	return db
}

// CleanupTestDB cleans up test data
func CleanupTestDB(t *testing.T, db *sql.DB) {
	// Delete all test data
	tables := []string{"trades", "portfolios", "users"}
	for _, table := range tables {
		_, err := db.Exec(fmt.Sprintf("DELETE FROM %s WHERE id > 0", table))
		if err != nil {
			log.Printf("Warning: Failed to cleanup table %s: %v", table, err)
		}
	}
}

// CreateTestUser creates a test user and returns user ID
func CreateTestUser(t *testing.T, db *sql.DB, username string, balance float64) int {
	var userID int

	// Make username unique by adding timestamp
	uniqueUsername := fmt.Sprintf("%s_%d", username, time.Now().UnixNano())

	err := db.QueryRow(
		"INSERT INTO users (username, email, cash_balance) VALUES ($1, $2, $3) RETURNING id",
		uniqueUsername, uniqueUsername+"@test.com", balance,
	).Scan(&userID)

	if err != nil {
		t.Fatalf("Failed to create test user: %v", err)
	}

	return userID
}
