package models

import "time"

// User represents a user in the system
type User struct {
	ID          int       `json:"id"`
	Username    string    `json:"username"`
	Email       string    `json:"email"`
	CashBalance float64   `json:"cash_balance"`
	CreatedAt   time.Time `json:"created_at"`
}

// Portfolio represents stocks owned by a user
type Portfolio struct {
	ID               int       `json:"id"`
	UserID           int       `json:"user_id"`
	StockSymbol      string    `json:"stock_symbol"`
	Quantity         int       `json:"quantity"`
	AvgPurchasePrice float64   `json:"avg_purchase_price"`
	UpdatedAt        time.Time `json:"updated_at"`
}

// Trade represents a buy/sell transaction
type Trade struct {
	ID          int       `json:"id"`
	UserID      int       `json:"user_id"`
	StockSymbol string    `json:"stock_symbol"`
	TradeType   string    `json:"trade_type"` // "BUY" or "SELL"
	Quantity    int       `json:"quantity"`
	Price       float64   `json:"price"`
	TotalAmount float64   `json:"total_amount"`
	Status      string    `json:"status"`
	CreatedAt   time.Time `json:"created_at"`
}

// BuyRequest - what client sends to buy stocks
type BuyRequest struct {
	UserID      int     `json:"user_id" binding:"required"`
	StockSymbol string  `json:"stock_symbol" binding:"required"`
	Quantity    int     `json:"quantity" binding:"required,min=1"`
	Price       float64 `json:"price" binding:"required,min=0.01"`
}

// PortfolioResponse - what we send back to client
type PortfolioResponse struct {
	Portfolio   []Portfolio `json:"portfolio"`
	CashBalance float64     `json:"cash_balance"`
	TotalValue  float64     `json:"total_value"`
}
