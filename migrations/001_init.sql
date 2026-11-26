-- Initial schema for Ayam Batu Inventory, Batch Pricing, Daily Sales, and Financial Reports

CREATE TABLE IF NOT EXISTS users (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(120) NOT NULL,
    email VARCHAR(180) NOT NULL UNIQUE,
    password_hash TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS products (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name VARCHAR(160) NOT NULL,
    sku VARCHAR(80) NOT NULL DEFAULT '',
    unit VARCHAR(40) NOT NULL DEFAULT 'pcs',
    description TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_products_user_id ON products(user_id);

CREATE TABLE IF NOT EXISTS stock_batches (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    product_id BIGINT NOT NULL REFERENCES products(id) ON DELETE CASCADE,
    batch_code VARCHAR(100) NOT NULL,
    buy_price NUMERIC(15, 2) NOT NULL CHECK (buy_price >= 0),
    initial_stock INTEGER NOT NULL CHECK (initial_stock > 0),
    sold_stock INTEGER NOT NULL DEFAULT 0 CHECK (sold_stock >= 0 AND sold_stock <= initial_stock),
    purchased_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    notes TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_stock_batches_user_id ON stock_batches(user_id);
CREATE INDEX IF NOT EXISTS idx_stock_batches_product_id ON stock_batches(product_id);
CREATE INDEX IF NOT EXISTS idx_stock_batches_purchased_at ON stock_batches(purchased_at);

CREATE TABLE IF NOT EXISTS sales (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    product_id BIGINT NOT NULL REFERENCES products(id) ON DELETE CASCADE,
    quantity INTEGER NOT NULL CHECK (quantity > 0),
    sell_price NUMERIC(15, 2) NOT NULL CHECK (sell_price >= 0),
    total_modal NUMERIC(15, 2) NOT NULL DEFAULT 0,
    total_bruto NUMERIC(15, 2) NOT NULL DEFAULT 0,
    total_netto NUMERIC(15, 2) NOT NULL DEFAULT 0,
    sold_at DATE NOT NULL DEFAULT CURRENT_DATE,
    notes TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_sales_user_id ON sales(user_id);
CREATE INDEX IF NOT EXISTS idx_sales_product_id ON sales(product_id);
CREATE INDEX IF NOT EXISTS idx_sales_sold_at ON sales(sold_at);

CREATE TABLE IF NOT EXISTS sale_batch_allocations (
    id BIGSERIAL PRIMARY KEY,
    sale_id BIGINT NOT NULL REFERENCES sales(id) ON DELETE CASCADE,
    batch_id BIGINT NOT NULL REFERENCES stock_batches(id) ON DELETE CASCADE,
    quantity INTEGER NOT NULL CHECK (quantity > 0),
    buy_price NUMERIC(15, 2) NOT NULL CHECK (buy_price >= 0),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_sale_batch_allocations_sale_id ON sale_batch_allocations(sale_id);
CREATE INDEX IF NOT EXISTS idx_sale_batch_allocations_batch_id ON sale_batch_allocations(batch_id);
