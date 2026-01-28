# Stock Trading Simulator

A high-performance stock trading simulator built with **Go**, **PostgreSQL**, and **WebSocket** for real-time price updates. Features concurrent order processing using goroutines and channels, demonstrating production-grade system design for financial applications.

![Go](https://img.shields.io/badge/Go-1.21-00ADD8?logo=go)
![PostgreSQL](https://img.shields.io/badge/PostgreSQL-15-336791?logo=postgresql)
![WebSocket](https://img.shields.io/badge/WebSocket-Real--time-orange)
![Docker](https://img.shields.io/badge/Docker-Ready-2496ED?logo=docker)

## ✨ Features

- **Concurrent Order Processing**: Worker pool pattern with 5 goroutines processing trades in parallel
- **Per-User Locking**: Fine-grained mutex locks ensuring thread safety while maximizing concurrency
- **Real-Time Price Updates**: WebSocket streaming for live stock prices
- **ACID Transactions**: PostgreSQL transactions with FOR UPDATE locks for data consistency
- **RESTful API**: Clean API design for trading operations
- **Responsive UI**: Simple HTML/CSS/JS frontend demonstrating WebSocket integration
- **Comprehensive Testing**: Unit tests with race condition detection and benchmarks

## 🏗️ Architecture

### Concurrency Model

The system uses a **worker pool pattern** with per-user locking:

- **5 concurrent workers** process trades from a buffered channel
- **Per-user mutex locks** allow different users to trade in parallel while ensuring same-user trades are sequential
- **Goroutines and channels** provide lightweight concurrent execution
- **Graceful shutdown** with sync.WaitGroup ensuring no in-flight trades are lost
```
Trade Request → Buffered Channel (queue) → Worker Pool (5 goroutines)
                                              ↓
                                    Per-User Mutex Lock
                                              ↓
                                    PostgreSQL Transaction
                                              ↓
                                         Trade Result
```

### Database Schema
```sql
users
├── id (SERIAL PRIMARY KEY)
├── username (VARCHAR UNIQUE)
├── email (VARCHAR UNIQUE)
└── cash_balance (DECIMAL)

portfolios
├── id (SERIAL PRIMARY KEY)
├── user_id (FK → users.id)
├── stock_symbol (VARCHAR)
├── quantity (INTEGER)
└── avg_purchase_price (DECIMAL)
    UNIQUE(user_id, stock_symbol)

trades
├── id (SERIAL PRIMARY KEY)
├── user_id (FK → users.id)
├── stock_symbol (VARCHAR)
├── trade_type (BUY/SELL)
├── quantity (INTEGER)
├── price (DECIMAL)
└── created_at (TIMESTAMP)
```

## 🚀 Quick Start

### Prerequisites

- Go 1.21+
- Docker & Docker Compose
- Git

### Installation
```bash
# Clone repository
git clone https://github.com/yourusername/stock-trading-simulator.git
cd stock-trading-simulator

# Copy environment file
cp .env.example .env

# Start PostgreSQL
docker-compose up -d

# Wait for database initialization (5 seconds)
sleep 5

# Run application
go run cmd/api/main.go
```

Open your browser: **http://localhost:8080**

## 📡 API Endpoints

### Trading Operations
```http
POST /api/trades/buy
POST /api/trades/sell
GET  /api/portfolio/:userId
GET  /api/trades/:userId
```

### WebSocket
```
ws://localhost:8080/ws/prices
```

### Example: Buy Stock
```bash
curl -X POST http://localhost:8080/api/trades/buy \
  -H "Content-Type: application/json" \
  -d '{
    "user_id": 1,
    "stock_symbol": "AAPL",
    "quantity": 10,
    "price": 150.50
  }'
```

**Response:**
```json
{
  "message": "Trade executed successfully",
  "trade_id": 42,
  "total_cost": 1505.00
}
```

## 🧪 Testing

### Run Tests
```bash
# All tests
go test ./internal/handlers -v

# With coverage
go test ./internal/handlers -cover

# Race condition detection
go test ./internal/handlers -race

# Benchmarks
go test ./internal/handlers -bench=. -benchmem
```

### Test Results
```
✅ 6/6 tests passing
✅ No race conditions detected
✅ 23% code coverage (focused on concurrent logic)
✅ Benchmark: 100+ trades/second throughput
```

## 🔧 Technical Decisions

### Why Per-User Locking?

**Initial approach:** Global mutex lock
-  Only 1 trade at a time across ALL users
-  No benefit from concurrent workers

**Current approach:** Per-user mutex locks
-  Different users trade in parallel
-  Same user's trades remain sequential (prevents race conditions)
-  Maximizes concurrency while maintaining correctness

### Why Worker Pool Pattern?

Instead of spawning a goroutine per request:
-  Bounded concurrency (predictable resource usage)
-  Backpressure via buffered channel (queue full = request waits)
-  Graceful shutdown possible

### Database Transaction Strategy

**Two-level safety:**
1. **Application lock** (mutex): Prevents concurrent portfolio logic
2. **Database lock** (FOR UPDATE): Prevents concurrent database modifications

**Why both?** Defense in depth for financial data integrity.

## 🛠️ Technology Stack

- **Backend**: Go 1.21
- **Database**: PostgreSQL 15
- **Real-time**: WebSocket (Gorilla WebSocket)
- **Web Framework**: Gin
- **Containerization**: Docker & Docker Compose
- **Frontend**: HTML, CSS, Vanilla JavaScript, Chart.js

## 📊 Performance Characteristics

- **Latency**: Sub-10ms order execution (in-memory matching)
- **Throughput**: 100+ simulated trades per second
- **Concurrency**: Handles 100+ concurrent users
- **Database**: Optimized with indexes on user_id and created_at

## 🎯 Key Learnings

This project demonstrates:

1. **Go Concurrency**: Goroutines, channels, mutexes, select statements
2. **Distributed Systems**: Worker pools, queuing, backpressure
3. **Financial Systems**: ACID transactions, idempotency, data consistency
4. **Real-time Communication**: WebSocket for bi-directional data flow
5. **Testing**: Race condition testing, benchmarking, edge cases


## 📄 License

MIT

## 👤 Author

**Atharva Konge**
- GitHub: [@atharvakonge](https://github.com/atharvakonge)

---

**Built as a learning project to explore Go's concurrency model and understand trading system architecture.**