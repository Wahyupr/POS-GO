-- =============================================================================
-- POS Backend - Initial Migration
-- =============================================================================

-- Enable UUID extension
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- =============================================================================
-- Merchants
-- =============================================================================
CREATE TYPE merchant_status AS ENUM ('ACTIVE', 'INACTIVE');

CREATE TABLE IF NOT EXISTS merchants (
    id         UUID         PRIMARY KEY DEFAULT uuid_generate_v4(),
    name       VARCHAR(150) NOT NULL,
    status     merchant_status NOT NULL DEFAULT 'ACTIVE',
    created_at TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

-- =============================================================================
-- Users
-- =============================================================================
CREATE TYPE user_role   AS ENUM ('ADMIN', 'MERCHANT', 'USER');
CREATE TYPE user_status AS ENUM ('ACTIVE', 'INACTIVE', 'PENDING');

CREATE TABLE IF NOT EXISTS users (
    id            UUID         PRIMARY KEY DEFAULT uuid_generate_v4(),
    email         VARCHAR(255) NOT NULL UNIQUE,
    username      VARCHAR(50)  UNIQUE,
    password_hash TEXT,
    google_id     VARCHAR(100) UNIQUE,
    google_avatar TEXT,
    name          VARCHAR(150) NOT NULL,
    role          user_role    NOT NULL DEFAULT 'USER',
    merchant_id   UUID         REFERENCES merchants(id) ON DELETE SET NULL,
    status        user_status  NOT NULL DEFAULT 'ACTIVE',
    created_at    TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_users_email      ON users(email);
CREATE INDEX IF NOT EXISTS idx_users_username   ON users(username);
CREATE INDEX IF NOT EXISTS idx_users_google_id  ON users(google_id);
CREATE INDEX IF NOT EXISTS idx_users_merchant   ON users(merchant_id);

-- =============================================================================
-- Refresh Tokens
-- =============================================================================
CREATE TABLE IF NOT EXISTS refresh_tokens (
    id         UUID         PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id    UUID         NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token_hash VARCHAR(64)  NOT NULL UNIQUE,
    expires_at TIMESTAMPTZ  NOT NULL,
    created_at TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_refresh_tokens_user   ON refresh_tokens(user_id);
CREATE INDEX IF NOT EXISTS idx_refresh_tokens_hash   ON refresh_tokens(token_hash);

-- =============================================================================
-- Products
-- =============================================================================
CREATE TYPE product_unit   AS ENUM ('PCS', 'KG', 'ONS', 'DUS');
CREATE TYPE product_status AS ENUM ('ACTIVE', 'INACTIVE');

CREATE TABLE IF NOT EXISTS products (
    id          UUID           PRIMARY KEY DEFAULT uuid_generate_v4(),
    merchant_id UUID           NOT NULL REFERENCES merchants(id) ON DELETE CASCADE,
    name        VARCHAR(200)   NOT NULL,
    image_url   TEXT,
    unit        product_unit   NOT NULL DEFAULT 'PCS',
    price_base  NUMERIC(12,2)  NOT NULL,
    stock       NUMERIC(12,3)  NOT NULL DEFAULT 0,
    status      product_status NOT NULL DEFAULT 'ACTIVE',
    created_at  TIMESTAMPTZ    NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_products_merchant ON products(merchant_id);

-- =============================================================================
-- Product Bulk Tiers
-- =============================================================================
CREATE TYPE pricing_mode AS ENUM ('UNIT_PRICE', 'BUNDLE_TOTAL');

CREATE TABLE IF NOT EXISTS product_bulk_tiers (
    id           UUID          PRIMARY KEY DEFAULT uuid_generate_v4(),
    product_id   UUID          NOT NULL REFERENCES products(id) ON DELETE CASCADE,
    min_qty      NUMERIC(12,3) NOT NULL,
    pricing_mode pricing_mode  NOT NULL,
    unit_price   NUMERIC(12,2),
    bundle_qty   NUMERIC(12,3),
    bundle_total NUMERIC(12,2)
);

CREATE INDEX IF NOT EXISTS idx_bulk_tiers_product ON product_bulk_tiers(product_id);

-- =============================================================================
-- Customers
-- =============================================================================
CREATE TABLE IF NOT EXISTS customers (
    id          UUID         PRIMARY KEY DEFAULT uuid_generate_v4(),
    merchant_id UUID         NOT NULL REFERENCES merchants(id) ON DELETE CASCADE,
    name        VARCHAR(150) NOT NULL,
    phone       VARCHAR(30),
    status      VARCHAR(20)  NOT NULL DEFAULT 'ACTIVE',
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_customers_merchant ON customers(merchant_id);

-- =============================================================================
-- Queues
-- =============================================================================
CREATE TYPE queue_status AS ENUM ('PENDING', 'PROCESS', 'DONE');

CREATE TABLE IF NOT EXISTS queues (
    id          UUID         PRIMARY KEY DEFAULT uuid_generate_v4(),
    merchant_id UUID         NOT NULL REFERENCES merchants(id) ON DELETE CASCADE,
    status      queue_status NOT NULL DEFAULT 'PENDING',
    notes       TEXT,
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

-- =============================================================================
-- Sales
-- =============================================================================
CREATE TYPE sale_status AS ENUM ('PAID', 'PARTIAL', 'DEBT');

CREATE TABLE IF NOT EXISTS sales (
    id          UUID          PRIMARY KEY DEFAULT uuid_generate_v4(),
    merchant_id UUID          NOT NULL REFERENCES merchants(id) ON DELETE CASCADE,
    customer_id UUID          REFERENCES customers(id) ON DELETE SET NULL,
    queue_id    UUID          REFERENCES queues(id) ON DELETE SET NULL,
    status      sale_status   NOT NULL DEFAULT 'PAID',
    total       NUMERIC(12,2) NOT NULL,
    discount    NUMERIC(12,2) NOT NULL DEFAULT 0,
    paid        NUMERIC(12,2) NOT NULL DEFAULT 0,
    change      NUMERIC(12,2) NOT NULL DEFAULT 0,
    created_at  TIMESTAMPTZ   NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_sales_merchant  ON sales(merchant_id);
CREATE INDEX IF NOT EXISTS idx_sales_customer  ON sales(customer_id);

-- =============================================================================
-- Sale Items
-- =============================================================================
CREATE TABLE IF NOT EXISTS sale_items (
    id                 UUID          PRIMARY KEY DEFAULT uuid_generate_v4(),
    sale_id            UUID          NOT NULL REFERENCES sales(id) ON DELETE CASCADE,
    product_id         UUID          NOT NULL REFERENCES products(id),
    qty                NUMERIC(12,3) NOT NULL,
    unit_price_applied NUMERIC(12,2) NOT NULL,
    line_total         NUMERIC(12,2) NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_sale_items_sale ON sale_items(sale_id);

-- =============================================================================
-- Payments
-- =============================================================================
CREATE TYPE payment_method AS ENUM ('CASH');

CREATE TABLE IF NOT EXISTS payments (
    id         UUID           PRIMARY KEY DEFAULT uuid_generate_v4(),
    sale_id    UUID           NOT NULL REFERENCES sales(id) ON DELETE CASCADE,
    amount     NUMERIC(12,2)  NOT NULL,
    method     payment_method NOT NULL DEFAULT 'CASH',
    created_at TIMESTAMPTZ    NOT NULL DEFAULT NOW()
);

-- =============================================================================
-- Inventory Movements
-- =============================================================================
CREATE TYPE inventory_reason AS ENUM ('SALE', 'ADJUSTMENT', 'RETURN');

CREATE TABLE IF NOT EXISTS inventory_movements (
    id         UUID             PRIMARY KEY DEFAULT uuid_generate_v4(),
    product_id UUID             NOT NULL REFERENCES products(id),
    change_qty NUMERIC(12,3)    NOT NULL,
    reason     inventory_reason NOT NULL,
    ref_id     UUID,
    created_at TIMESTAMPTZ      NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_inventory_product ON inventory_movements(product_id);
