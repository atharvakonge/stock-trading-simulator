package handlers

import (
	// "encoding/json"
	"log"
	"math/rand"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

// PriceUpdate represents a stock price update
type PriceUpdate struct {
	Symbol    string    `json:"symbol"`
	Price     float64   `json:"price"`
	Change    float64   `json:"change"`
	Timestamp time.Time `json:"timestamp"`
}

// WebSocket upgrader
var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all origins (for development and demo)
	},
}

// HandleWebSocket handles WebSocket connections for price updates
func HandleWebSocket(c *gin.Context) {
	// Upgrade HTTP connection to WebSocket
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Println("WebSocket upgrade error:", err)
		return
	}
	defer conn.Close()

	log.Println("Client connected to WebSocket")

	// Stock symbols to simulate
	symbols := []string{"AAPL", "GOOGL", "MSFT", "TSLA", "AMZN"}

	// Initial prices
	prices := map[string]float64{
		"AAPL":  150.00,
		"GOOGL": 140.00,
		"MSFT":  380.00,
		"TSLA":  250.00,
		"AMZN":  180.00,
	}

	// Send price updates every second
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			// Pick random stock
			symbol := symbols[rand.Intn(len(symbols))]

			// Simulate price change (-2% to +2%)
			changePercent := (rand.Float64() - 0.5) * 4
			oldPrice := prices[symbol]
			newPrice := oldPrice * (1 + changePercent/100)
			prices[symbol] = newPrice

			// Create update
			update := PriceUpdate{
				Symbol:    symbol,
				Price:     newPrice,
				Change:    changePercent,
				Timestamp: time.Now(),
			}

			// Send to client
			if err := conn.WriteJSON(update); err != nil {
				log.Println("WebSocket write error:", err)
				return
			}

			log.Printf("Sent price update: %s = $%.2f (%.2f%%)",
				symbol, newPrice, changePercent)
		}
	}
}
