-- Stock Trading Simulator - Database Schema
-- Auto-runs on first container startup

-- Users table
CREATE TABLE IF NOT EXISTS users (
    id SERIAL PRIMARY KEY,
    username VARCHAR(100) UNIQUE NOT NULL,
    email VARCHAR(255) UNIQUE NOT NULL,
    cash_balance DECIMAL(15,2) DEFAULT 10000.00,
    created_at TIMESTAMP DEFAULT NOW()
);

-- Portfolios table (current holdings)
CREATE TABLE IF NOT EXISTS portfolios (
    id SERIAL PRIMARY KEY,
    user_id INTEGER REFERENCES users(id) ON DELETE CASCADE,
    stock_symbol VARCHAR(10) NOT NULL,
    quantity INTEGER NOT NULL CHECK (quantity >= 0),
    avg_purchase_price DECIMAL(10,2) NOT NULL,
    updated_at TIMESTAMP DEFAULT NOW(),
    UNIQUE(user_id, stock_symbol)
);

-- Trades table (transaction history)
CREATE TABLE IF NOT EXISTS trades (
    id SERIAL PRIMARY KEY,
    user_id INTEGER REFERENCES users(id) ON DELETE CASCADE,
    stock_symbol VARCHAR(10) NOT NULL,
    trade_type VARCHAR(4) NOT NULL CHECK (trade_type IN ('BUY', 'SELL')),
    quantity INTEGER NOT NULL CHECK (quantity > 0),
    price DECIMAL(10,2) NOT NULL,
    total_amount DECIMAL(15,2) NOT NULL,
    status VARCHAR(20) DEFAULT 'COMPLETED',
    created_at TIMESTAMP DEFAULT NOW()
);

-- Indexes for query performance
CREATE INDEX IF NOT EXISTS idx_trades_user_id ON trades(user_id);
CREATE INDEX IF NOT EXISTS idx_trades_created_at ON trades(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_portfolios_user_id ON portfolios(user_id);

-- Function to limit trades per user to 15 most recent
CREATE OR REPLACE FUNCTION limit_user_trades()
RETURNS TRIGGER AS $$
BEGIN
    -- Delete oldest trades if user has more than 15
    DELETE FROM trades
    WHERE id IN (
        SELECT id FROM trades
        WHERE user_id = NEW.user_id
        ORDER BY created_at DESC
        OFFSET 15
    );
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Trigger to automatically enforce trade limit
DROP TRIGGER IF EXISTS enforce_trade_limit ON trades;
CREATE TRIGGER enforce_trade_limit
AFTER INSERT ON trades
FOR EACH ROW
EXECUTE FUNCTION limit_user_trades();

-- Demo user for testing
INSERT INTO users (id, username, email, cash_balance) 
VALUES (1, 'demo_user', 'demo@example.com', 10000.00)
ON CONFLICT (id) DO UPDATE 
SET cash_balance = EXCLUDED.cash_balance;

-- Success message
SELECT 'Database initialized successfully with trade limit trigger!' as status;