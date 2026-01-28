#!/bin/bash

echo "Testing concurrent trades for DIFFERENT users (should be parallel)..."

# User 1 - 5 trades
for i in {1..5}; do
    curl -X POST http://localhost:8080/api/trades/buy \
      -H "Content-Type: application/json" \
      -d '{
        "user_id": 1,
        "stock_symbol": "AAPL",
        "quantity": 1,
        "price": 150.00
      }' &
done

# User 2 - 5 trades (should run in parallel with User 1!)
for i in {1..5}; do
    curl -X POST http://localhost:8080/api/trades/buy \
      -H "Content-Type: application/json" \
      -d '{
        "user_id": 2,
        "stock_symbol": "GOOGL",
        "quantity": 1,
        "price": 140.00
      }' &
done

# User 3 - 5 trades (should run in parallel with Users 1 & 2!)
for i in {1..5}; do
    curl -X POST http://localhost:8080/api/trades/buy \
      -H "Content-Type: application/json" \
      -d '{
        "user_id": 3,
        "stock_symbol": "MSFT",
        "quantity": 1,
        "price": 380.00
      }' &
done

wait
echo ""
echo "All concurrent requests completed!"
echo "Check server logs to see different workers processing different users in parallel"