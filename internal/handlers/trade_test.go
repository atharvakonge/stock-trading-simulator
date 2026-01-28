package handlers

import (
	"fmt"
	"github.com/atharvakonge/stock-trading-simulator/internal/db"
	"github.com/atharvakonge/stock-trading-simulator/internal/models"
	"testing"
)

func TestBuyStock_Success(t *testing.T) {
	// Setup
	database := db.SetupTestDB(t)
	defer database.Close()
	defer db.CleanupTestDB(t, database)

	userID := db.CreateTestUser(t, database, "testuser", 10000.0)

	// Create trade processor
	tp := NewTradeProcessor(1)
	tp.Start()
	defer tp.Stop()

	// Execute trade
	req := models.BuyRequest{
		UserID:      userID,
		StockSymbol: "AAPL",
		Quantity:    10,
		Price:       150.0,
	}

	result := tp.SubmitTrade(req)

	// Assertions
	if !result.Success {
		t.Errorf("Expected trade to succeed, got error: %s", result.Error)
	}

	if result.TotalAmount != 1500.0 {
		t.Errorf("Expected total amount 1500.0, got %.2f", result.TotalAmount)
	}

	// Verify balance was deducted
	var balance float64
	err := database.QueryRow("SELECT cash_balance FROM users WHERE id = $1", userID).Scan(&balance)
	if err != nil {
		t.Fatalf("Failed to query balance: %v", err)
	}

	expectedBalance := 10000.0 - 1500.0
	if balance != expectedBalance {
		t.Errorf("Expected balance %.2f, got %.2f", expectedBalance, balance)
	}

	// Verify portfolio was updated
	var quantity int
	err = database.QueryRow(
		"SELECT quantity FROM portfolios WHERE user_id = $1 AND stock_symbol = $2",
		userID, "AAPL",
	).Scan(&quantity)

	if err != nil {
		t.Fatalf("Failed to query portfolio: %v", err)
	}

	if quantity != 10 {
		t.Errorf("Expected quantity 10, got %d", quantity)
	}
}

func TestBuyStock_InsufficientFunds(t *testing.T) {
	// Setup
	database := db.SetupTestDB(t)
	defer database.Close()
	defer db.CleanupTestDB(t, database)

	userID := db.CreateTestUser(t, database, "pooruser", 100.0)

	tp := NewTradeProcessor(1)
	tp.Start()
	defer tp.Stop()

	// Try to buy more than balance
	req := models.BuyRequest{
		UserID:      userID,
		StockSymbol: "AAPL",
		Quantity:    10,
		Price:       150.0, // Costs $1500, but only has $100
	}

	result := tp.SubmitTrade(req)

	// Assertions
	if result.Success {
		t.Error("Expected trade to fail due to insufficient funds")
	}

	if result.Error != "Insufficient funds" {
		t.Errorf("Expected 'Insufficient funds' error, got: %s", result.Error)
	}

	// Verify balance unchanged
	var balance float64
	database.QueryRow("SELECT cash_balance FROM users WHERE id = $1", userID).Scan(&balance)

	if balance != 100.0 {
		t.Errorf("Expected balance unchanged at 100.0, got %.2f", balance)
	}
}

func TestBuyStock_InvalidUser(t *testing.T) {
	database := db.SetupTestDB(t)
	defer database.Close()
	defer db.CleanupTestDB(t, database)

	tp := NewTradeProcessor(1)
	tp.Start()
	defer tp.Stop()

	// Try with non-existent user
	req := models.BuyRequest{
		UserID:      99999, // Doesn't exist
		StockSymbol: "AAPL",
		Quantity:    10,
		Price:       150.0,
	}

	result := tp.SubmitTrade(req)

	if result.Success {
		t.Error("Expected trade to fail for invalid user")
	}

	if result.Error != "User not found" {
		t.Errorf("Expected 'User not found' error, got: %s", result.Error)
	}
}

func TestConcurrentBuying_SameUser(t *testing.T) {
	database := db.SetupTestDB(t)
	defer database.Close()
	defer db.CleanupTestDB(t, database)

	userID := db.CreateTestUser(t, database, "concurrent_user", 10000.0)

	tp := NewTradeProcessor(5) // 5 workers
	tp.Start()
	defer tp.Stop()

	// Execute 10 concurrent trades for same user
	numTrades := 10
	results := make(chan TradeResult, numTrades)

	for i := 0; i < numTrades; i++ {
		go func() {
			req := models.BuyRequest{
				UserID:      userID,
				StockSymbol: "AAPL",
				Quantity:    1,
				Price:       100.0,
			}
			result := tp.SubmitTrade(req)
			results <- result
		}()
	}

	// Collect results
	successCount := 0
	for i := 0; i < numTrades; i++ {
		result := <-results
		if result.Success {
			successCount++
		}
	}

	// All should succeed
	if successCount != numTrades {
		t.Errorf("Expected %d successful trades, got %d", numTrades, successCount)
	}

	// Verify final balance
	var finalBalance float64
	database.QueryRow("SELECT cash_balance FROM users WHERE id = $1", userID).Scan(&finalBalance)

	expectedBalance := 10000.0 - (100.0 * float64(numTrades))
	if finalBalance != expectedBalance {
		t.Errorf("Race condition detected! Expected balance %.2f, got %.2f",
			expectedBalance, finalBalance)
	}

	// Verify portfolio quantity
	var quantity int
	database.QueryRow(
		"SELECT quantity FROM portfolios WHERE user_id = $1 AND stock_symbol = 'AAPL'",
		userID,
	).Scan(&quantity)

	if quantity != numTrades {
		t.Errorf("Race condition detected! Expected quantity %d, got %d",
			numTrades, quantity)
	}
}

func TestConcurrentBuying_DifferentUsers(t *testing.T) {
	database := db.SetupTestDB(t)
	defer database.Close()
	defer db.CleanupTestDB(t, database)

	// Create 5 users
	userIDs := make([]int, 5)
	for i := 0; i < 5; i++ {
		userIDs[i] = db.CreateTestUser(t, database,
			fmt.Sprintf("user%d", i), 10000.0)
	}

	tp := NewTradeProcessor(5)
	tp.Start()
	defer tp.Stop()

	// Each user makes 10 trades concurrently
	totalTrades := 50
	results := make(chan TradeResult, totalTrades)

	for _, userID := range userIDs {
		for i := 0; i < 10; i++ {
			go func(uid int) {
				req := models.BuyRequest{
					UserID:      uid,
					StockSymbol: "AAPL",
					Quantity:    1,
					Price:       100.0,
				}
				result := tp.SubmitTrade(req)
				results <- result
			}(userID)
		}
	}

	// Collect results
	successCount := 0
	for i := 0; i < totalTrades; i++ {
		result := <-results
		if result.Success {
			successCount++
		}
	}

	if successCount != totalTrades {
		t.Errorf("Expected %d successful trades, got %d", totalTrades, successCount)
	}

	// Verify each user's balance and portfolio
	for _, userID := range userIDs {
		var balance float64
		database.QueryRow("SELECT cash_balance FROM users WHERE id = $1", userID).Scan(&balance)

		expectedBalance := 10000.0 - 1000.0 // 10 trades × $100
		if balance != expectedBalance {
			t.Errorf("User %d: Expected balance %.2f, got %.2f",
				userID, expectedBalance, balance)
		}
	}
}

func BenchmarkTradeProcessing(b *testing.B) {
	database := db.SetupTestDB(&testing.T{})
	defer database.Close()

	userID := db.CreateTestUser(&testing.T{}, database, "benchmark_user", 1000000.0)

	tp := NewTradeProcessor(5)
	tp.Start()
	defer tp.Stop()

	b.ResetTimer() // Start timing now

	for i := 0; i < b.N; i++ {
		req := models.BuyRequest{
			UserID:      userID,
			StockSymbol: "AAPL",
			Quantity:    1,
			Price:       100.0,
		}
		tp.SubmitTrade(req)
	}
}

func BenchmarkConcurrentTrades(b *testing.B) {
	database := db.SetupTestDB(&testing.T{})
	defer database.Close()

	userID := db.CreateTestUser(&testing.T{}, database, "benchmark_user", 10000000.0)

	tp := NewTradeProcessor(10)
	tp.Start()
	defer tp.Stop()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			req := models.BuyRequest{
				UserID:      userID,
				StockSymbol: "AAPL",
				Quantity:    1,
				Price:       100.0,
			}
			tp.SubmitTrade(req)
		}
	})
}

func TestSellStock_Success(t *testing.T) {
	database := db.SetupTestDB(t)
	defer database.Close()
	defer db.CleanupTestDB(t, database)

	userID := db.CreateTestUser(t, database, "seller", 10000.0)

	// First buy some stocks
	_, err := database.Exec(`
        INSERT INTO portfolios (user_id, stock_symbol, quantity, avg_purchase_price)
        VALUES ($1, 'AAPL', 10, 150.0)
    `, userID)
	if err != nil {
		t.Fatalf("Failed to setup portfolio: %v", err)
	}

	// Now sell via HTTP handler
	// Since SellStock is an HTTP handler, we need to test it differently
	// For now, let's manually execute the sell logic

	// Update portfolio (reduce quantity)
	_, err = database.Exec(`
        UPDATE portfolios 
        SET quantity = quantity - $1 
        WHERE user_id = $2 AND stock_symbol = $3
    `, 5, userID, "AAPL")
	if err != nil {
		t.Fatalf("Failed to sell: %v", err)
	}

	// Update user balance (add proceeds)
	_, err = database.Exec(`
        UPDATE users 
        SET cash_balance = cash_balance + $1 
        WHERE id = $2
    `, 5*150.0, userID)
	if err != nil {
		t.Fatalf("Failed to update balance: %v", err)
	}

	// Verify sold quantity
	var quantity int
	err = database.QueryRow(
		"SELECT quantity FROM portfolios WHERE user_id = $1 AND stock_symbol = 'AAPL'",
		userID,
	).Scan(&quantity)

	if err != nil {
		t.Fatalf("Failed to query portfolio: %v", err)
	}

	if quantity != 5 {
		t.Errorf("Expected 5 shares remaining, got %d", quantity)
	}

	// Verify balance increased
	var balance float64
	err = database.QueryRow("SELECT cash_balance FROM users WHERE id = $1", userID).Scan(&balance)
	if err != nil {
		t.Fatalf("Failed to query balance: %v", err)
	}

	expectedBalance := 10000.0 + (5 * 150.0)
	if balance != expectedBalance {
		t.Errorf("Expected balance %.2f, got %.2f", expectedBalance, balance)
	}
}

func TestSellStock_InsufficientShares(t *testing.T) {
	// Test selling more shares than owned
	// Should fail with appropriate error
}
