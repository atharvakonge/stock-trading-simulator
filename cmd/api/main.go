package main

import (
	"log"
	"os"

	"github.com/atharvakonge/stock-trading-simulator/internal/db"
	"github.com/atharvakonge/stock-trading-simulator/internal/handlers"
	"github.com/atharvakonge/stock-trading-simulator/internal/models"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

func main() {
	// Load .env file
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using defaults or environment variables")
	}

	// Initialize database
	if err := db.InitDB(); err != nil {
		log.Fatal("Failed to connect to database:", err)
	}
	defer db.CloseDB()

	// Get number of workers from env or default to 5
	numWorkers := 5
	if workers := os.Getenv("NUM_WORKERS"); workers != "" {
		// Parse workers string to int if needed
		numWorkers = 5 // For simplicity, keeping default
	}

	// Initialize trade processor
	tradeProcessor := handlers.NewTradeProcessor(numWorkers)
	tradeProcessor.Start()
	defer tradeProcessor.Stop()

	// Set Gin mode based on environment
	if os.Getenv("GIN_MODE") == "release" {
		gin.SetMode(gin.ReleaseMode)
	}

	// Create Gin router
	router := gin.Default()

	// API routes
	api := router.Group("/api")
	{
		// Trading endpoints
		api.POST("/trades/buy", func(c *gin.Context) {
			var req models.BuyRequest
			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(400, gin.H{"error": err.Error()})
				return
			}

			result := tradeProcessor.SubmitTrade(req)
			if !result.Success {
				c.JSON(400, gin.H{"error": result.Error})
				return
			}

			c.JSON(200, gin.H{
				"message":    "Trade executed successfully",
				"trade_id":   result.TradeID,
				"total_cost": result.TotalAmount,
			})
		})

		api.POST("/trades/sell", handlers.SellStock)
		api.GET("/trades/:userId", handlers.GetTradeHistory)
		api.GET("/portfolio/:userId", handlers.GetPortfolio)
	}

	// WebSocket endpoint
	router.GET("/ws/prices", handlers.HandleWebSocket)

	// Health check
	router.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "healthy"})
	})

	// Serve frontend
	router.GET("/", func(c *gin.Context) {
		c.File("./public/index.html")
	})

	router.NoRoute(func(c *gin.Context) {
		c.File("./public/index.html")
	})

	// Get port from environment or default
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Println("🚀 Server starting on http://localhost:" + port)
	log.Println("📊 Open http://localhost:" + port + " in your browser")

	if err := router.Run(":" + port); err != nil {
		log.Fatal("Failed to start server:", err)
	}
}
