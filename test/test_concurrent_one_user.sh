#!/bin/bash

# Buy AAPL 10 times concurrently
for i in {1..10}; do
    curl -X POST http://localhost:8080/api/trades/buy \
      -H "Content-Type: application/json" \
      -d '{
        "user_id": 1,
        "stock_symbol": "AAPL",
        "quantity": 5,
        "price": 150.00
      }' &
done

wait
echo "All requests completed!"