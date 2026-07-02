-- Create users table
CREATE TABLE users (
    id SERIAL PRIMARY KEY,
    username TEXT UNIQUE NOT NULL,
    password_hash TEXT NOT NULL,
    role TEXT NOT NULL CHECK (role IN ('admin', 'cashier')),
    created_at TIMESTAMP DEFAULT NOW()
);

-- Create product_groups table
CREATE TABLE product_groups (
    id SERIAL PRIMARY KEY,
    name TEXT NOT NULL,
    description TEXT,
    created_at TIMESTAMP DEFAULT NOW()
);

-- Create products table
CREATE TABLE products (
    id SERIAL PRIMARY KEY,
    name TEXT NOT NULL,
    sku TEXT UNIQUE NOT NULL,
    price INTEGER NOT NULL CHECK (price >= 0),
    stock INTEGER NOT NULL CHECK (stock >= 0),
    group_id INTEGER REFERENCES product_groups(id),
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),
    deleted_at TIMESTAMP
);

-- Create sales table
CREATE TABLE sales (
    id SERIAL PRIMARY KEY,
    total_amount INTEGER NOT NULL CHECK (total_amount >= 0),
    payment_method TEXT NOT NULL,
    cashier_id INTEGER REFERENCES users(id),
    created_at TIMESTAMP DEFAULT NOW()
);

-- Create sale_items table
CREATE TABLE sale_items (
    id SERIAL PRIMARY KEY,
    sale_id INTEGER REFERENCES sales(id) ON DELETE CASCADE,
    product_id INTEGER REFERENCES products(id),
    quantity INTEGER NOT NULL CHECK (quantity > 0),
    price_at_sale INTEGER NOT NULL CHECK (price_at_sale >= 0)
);

-- Create index on SKU for faster lookup
CREATE INDEX idx_products_sku ON products(sku);
