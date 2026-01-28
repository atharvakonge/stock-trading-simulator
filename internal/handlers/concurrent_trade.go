package handlers

import (
	"database/sql"
	"log"
	"sync"

	"github.com/atharvakonge/stock-trading-simulator/internal/db"
	"github.com/atharvakonge/stock-trading-simulator/internal/models"
)

// TradeResult represents result of a trade operation
type TradeResult struct {
	TradeID     int
	Success     bool
	Error       string
	TotalAmount float64
}

// TradeRequest represents a trade to be processed
type TradeRequest struct {
	Request  models.BuyRequest
	ResultCh chan TradeResult // Channel to send result back
}

// TradeProcessor handles concurrent trade processing
type TradeProcessor struct {
	workers      int
	tradeQueue   chan TradeRequest
	stopCh       chan struct{}
	wg           sync.WaitGroup
	portfolioMgr *models.PortfolioManager
}

// NewTradeProcessor creates a new trade processor with worker pool
func NewTradeProcessor(workers int) *TradeProcessor {
	return &TradeProcessor{
		workers:      workers,
		tradeQueue:   make(chan TradeRequest, 100), // Buffer of 100 trades
		stopCh:       make(chan struct{}),
		portfolioMgr: models.NewPortfolioManager(),
	}
}

// Start starts the worker pool
func (tp *TradeProcessor) Start() {
	for i := 0; i < tp.workers; i++ {
		tp.wg.Add(1)
		go tp.worker(i)
	}
	log.Printf("✅ Started %d trade workers", tp.workers)
}

// Stop gracefully stops all workers
func (tp *TradeProcessor) Stop() {
	close(tp.stopCh)
	tp.wg.Wait()
	log.Println("Trade processor stopped")
}

// worker processes trades from the queue
func (tp *TradeProcessor) worker(id int) {
	defer tp.wg.Done()

	log.Printf("Worker %d started", id)

	for {
		select {
		case <-tp.stopCh:
			log.Printf("Worker %d stopping", id)
			return

		case tradeReq := <-tp.tradeQueue:
			log.Printf("Worker %d processing trade for User %d: %s x%d",
				id, tradeReq.Request.UserID, tradeReq.Request.StockSymbol, tradeReq.Request.Quantity)

			result := tp.processTrade(tradeReq.Request)
			tradeReq.ResultCh <- result
		}
	}
}

// processTrade executes a single trade with per-user locking
func (tp *TradeProcessor) processTrade(req models.BuyRequest) TradeResult {
	// Lock portfolio for THIS USER ONLY (not global!)
	tp.portfolioMgr.LockUser(req.UserID)
	defer tp.portfolioMgr.UnlockUser(req.UserID)

	// Start database transaction
	tx, err := db.DB.Begin()
	if err != nil {
		return TradeResult{Success: false, Error: "Transaction failed"}
	}
	defer tx.Rollback()

	totalCost := req.Price * float64(req.Quantity)

	// 1. Check user has enough cash
	var cashBalance float64
	err = tx.QueryRow(
		"SELECT cash_balance FROM users WHERE id = $1 FOR UPDATE",
		req.UserID,
	).Scan(&cashBalance)

	if err == sql.ErrNoRows {
		return TradeResult{Success: false, Error: "User not found"}
	}
	if err != nil {
		return TradeResult{Success: false, Error: "Database error"}
	}

	if cashBalance < totalCost {
		return TradeResult{Success: false, Error: "Insufficient funds"}
	}

	// 2. Deduct cash
	_, err = tx.Exec(
		"UPDATE users SET cash_balance = cash_balance - $1 WHERE id = $2",
		totalCost, req.UserID,
	)
	if err != nil {
		return TradeResult{Success: false, Error: "Failed to update balance"}
	}

	// 3. Update portfolio
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
		return TradeResult{Success: false, Error: "Failed to update portfolio"}
	}

	// 4. Record trade
	var tradeID int
	err = tx.QueryRow(`
        INSERT INTO trades (user_id, stock_symbol, trade_type, quantity, price, total_amount)
        VALUES ($1, $2, 'BUY', $3, $4, $5)
        RETURNING id
    `, req.UserID, req.StockSymbol, req.Quantity, req.Price, totalCost).Scan(&tradeID)

	if err != nil {
		return TradeResult{Success: false, Error: "Failed to record trade"}
	}

	// Commit transaction
	if err = tx.Commit(); err != nil {
		return TradeResult{Success: false, Error: "Transaction commit failed"}
	}

	log.Printf("Worker completed trade %d for User %d", tradeID, req.UserID)

	return TradeResult{
		TradeID:     tradeID,
		Success:     true,
		TotalAmount: totalCost,
	}
}

// SubmitTrade submits a trade to the processing queue
func (tp *TradeProcessor) SubmitTrade(req models.BuyRequest) TradeResult {
	resultCh := make(chan TradeResult)

	// Send trade to queue
	tp.tradeQueue <- TradeRequest{
		Request:  req,
		ResultCh: resultCh,
	}

	// Wait for result
	result := <-resultCh
	return result
}
