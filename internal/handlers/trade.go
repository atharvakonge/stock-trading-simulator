package handlers

import (
	"database/sql"
	"fmt"
	"net/http"

	"github.com/atharvakonge/stock-trading-simulator/internal/db"
	"github.com/atharvakonge/stock-trading-simulator/internal/models"
	"github.com/gin-gonic/gin"
)

// BuyStock handles POST /api/trades/buy
func BuyStock(c *gin.Context) {
	var req models.BuyRequest

	// Parse JSON request body
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Start database transaction
	tx, err := db.DB.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Transaction failed"})
		return
	}
	defer tx.Rollback() // Rollback if we don't commit

	// Calculate total cost
	totalCost := req.Price * float64(req.Quantity)

	// 1. Check user has enough cash
	var cashBalance float64
	err = tx.QueryRow(
		"SELECT cash_balance FROM users WHERE id = $1 FOR UPDATE",
		req.UserID,
	).Scan(&cashBalance)

	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}

	if cashBalance < totalCost {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Insufficient funds"})
		return
	}

	// 2. Deduct cash from user
	_, err = tx.Exec(
		"UPDATE users SET cash_balance = cash_balance - $1 WHERE id = $2",
		totalCost, req.UserID,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update balance"})
		return
	}

	// 3. Update portfolio (or insert if doesn't exist)
	_, err = tx.Exec(`
        INSERT INTO portfolios (user_id, stock_symbol, quantity, avg_purchase_price)
        VALUES ($1, $2, $3, $4)
        ON CONFLICT (user_id, stock_symbol) 
        DO UPDATE SET 
            quantity = portfolios.quantity + $3,
            avg_purchase_price = (
                (portfolios.avg_purchase_price * portfolios.quantity) + ($4 * $3)
            ) / (portfolios.quantity + $3),
            updated_at = NOW()
    `, req.UserID, req.StockSymbol, req.Quantity, req.Price)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update portfolio"})
		return
	}

	// 4. Record trade in history
	var tradeID int
	err = tx.QueryRow(`
        INSERT INTO trades (user_id, stock_symbol, trade_type, quantity, price, total_amount)
        VALUES ($1, $2, 'BUY', $3, $4, $5)
        RETURNING id
    `, req.UserID, req.StockSymbol, req.Quantity, req.Price, totalCost).Scan(&tradeID)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to record trade"})
		return
	}

	// Commit transaction (all or nothing!)
	if err = tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Transaction commit failed"})
		return
	}

	// Success!
	c.JSON(http.StatusOK, gin.H{
		"message":     "Trade executed successfully",
		"trade_id":    tradeID,
		"total_cost":  totalCost,
		"new_balance": cashBalance - totalCost,
	})
}

// GetPortfolio handles GET /api/portfolio/:userId
func GetPortfolio(c *gin.Context) {
	userID := c.Param("userId")

	// Get user's cash balance
	var cashBalance float64
	err := db.DB.QueryRow(
		"SELECT cash_balance FROM users WHERE id = $1",
		userID,
	).Scan(&cashBalance)

	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}

	// Get user's portfolio
	rows, err := db.DB.Query(`
        SELECT id, user_id, stock_symbol, quantity, avg_purchase_price, updated_at
        FROM portfolios
        WHERE user_id = $1 AND quantity > 0
        ORDER BY stock_symbol
    `, userID)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch portfolio"})
		return
	}
	defer rows.Close()

	portfolio := make([]models.Portfolio, 0)
	totalValue := cashBalance // Start with cash

	for rows.Next() {
		var p models.Portfolio
		err := rows.Scan(&p.ID, &p.UserID, &p.StockSymbol, &p.Quantity, &p.AvgPurchasePrice, &p.UpdatedAt)
		if err != nil {
			continue
		}
		portfolio = append(portfolio, p)
		// For now, use avg purchase price as "current value"
		totalValue += p.AvgPurchasePrice * float64(p.Quantity)
	}

	c.JSON(http.StatusOK, models.PortfolioResponse{
		Portfolio:   portfolio,
		CashBalance: cashBalance,
		TotalValue:  totalValue,
	})
}

// SellStock handles POST /api/trades/sell
func SellStock(c *gin.Context) {
	var req models.BuyRequest // Reuse same struct

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	tx, err := db.DB.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Transaction failed"})
		return
	}
	defer tx.Rollback()

	totalProceeds := req.Price * float64(req.Quantity)

	// 1. Check user owns enough shares
	var currentQuantity int
	err = tx.QueryRow(
		"SELECT quantity FROM portfolios WHERE user_id = $1 AND stock_symbol = $2 FOR UPDATE",
		req.UserID, req.StockSymbol,
	).Scan(&currentQuantity)

	if err == sql.ErrNoRows {
		c.JSON(http.StatusBadRequest, gin.H{"error": "You don't own this stock"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}

	if currentQuantity < req.Quantity {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": fmt.Sprintf("Insufficient shares. You own %d, trying to sell %d",
				currentQuantity, req.Quantity),
		})
		return
	}

	// 2. Update portfolio (reduce quantity)
	newQuantity := currentQuantity - req.Quantity
	if newQuantity == 0 {
		// Delete portfolio entry if selling all
		_, err = tx.Exec(
			"DELETE FROM portfolios WHERE user_id = $1 AND stock_symbol = $2",
			req.UserID, req.StockSymbol,
		)
	} else {
		// Update quantity
		_, err = tx.Exec(
			"UPDATE portfolios SET quantity = $1, updated_at = NOW() WHERE user_id = $2 AND stock_symbol = $3",
			newQuantity, req.UserID, req.StockSymbol,
		)
	}

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update portfolio"})
		return
	}

	// 3. Add proceeds to user's cash
	_, err = tx.Exec(
		"UPDATE users SET cash_balance = cash_balance + $1 WHERE id = $2",
		totalProceeds, req.UserID,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update balance"})
		return
	}

	// 4. Record trade
	var tradeID int
	err = tx.QueryRow(`
        INSERT INTO trades (user_id, stock_symbol, trade_type, quantity, price, total_amount)
        VALUES ($1, $2, 'SELL', $3, $4, $5)
        RETURNING id
    `, req.UserID, req.StockSymbol, req.Quantity, req.Price, totalProceeds).Scan(&tradeID)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to record trade"})
		return
	}

	if err = tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Transaction commit failed"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":        "Stock sold successfully",
		"trade_id":       tradeID,
		"total_proceeds": totalProceeds,
	})
}

// GetTradeHistory handles GET /api/trades/:userId
func GetTradeHistory(c *gin.Context) {
	userID := c.Param("userId")

	rows, err := db.DB.Query(`
        SELECT id, stock_symbol, trade_type, quantity, price, total_amount, created_at
        FROM trades
        WHERE user_id = $1
        ORDER BY created_at DESC
        LIMIT 50
    `, userID)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch trades"})
		return
	}
	defer rows.Close()

	var trades []models.Trade
	for rows.Next() {
		var t models.Trade
		err := rows.Scan(&t.ID, &t.StockSymbol, &t.TradeType, &t.Quantity,
			&t.Price, &t.TotalAmount, &t.CreatedAt)
		if err != nil {
			continue
		}
		trades = append(trades, t)
	}

	c.JSON(http.StatusOK, gin.H{
		"trades": trades,
		"count":  len(trades),
	})
}
