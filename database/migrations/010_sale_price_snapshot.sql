BEGIN;

-- Cart sessions: server-side active transactions (open / held / checked_out / cancelled / expired)
CREATE TABLE IF NOT EXISTS cart_sessions (
    id           SERIAL PRIMARY KEY,
    cashier_id   INTEGER NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    store_id     INTEGER REFERENCES stores(id) ON DELETE SET NULL,
    shift_id     INTEGER REFERENCES shifts(id),
    customer_id  INTEGER REFERENCES customers(id),
    status       VARCHAR(20) NOT NULL DEFAULT 'open'
                 CHECK (status IN ('open', 'held', 'checked_out', 'cancelled', 'expired')),
    subtotal     INTEGER NOT NULL DEFAULT 0,
    discount     INTEGER NOT NULL DEFAULT 0,
    tax          INTEGER NOT NULL DEFAULT 0,
    total_amount INTEGER NOT NULL DEFAULT 0,
    expired_at   TIMESTAMPTZ,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_cart_sessions_cashier_status ON cart_sessions(cashier_id, status);
CREATE INDEX IF NOT EXISTS idx_cart_sessions_shift ON cart_sessions(shift_id);

-- One open cart per cashier at a time (partial unique index)
CREATE UNIQUE INDEX IF NOT EXISTS uq_cart_sessions_open_cashier
    ON cart_sessions(cashier_id) WHERE status = 'open';

-- Cart items: immutable price snapshot per line item
CREATE TABLE IF NOT EXISTS cart_items (
    id                  SERIAL PRIMARY KEY,
    cart_session_id     INTEGER NOT NULL REFERENCES cart_sessions(id) ON DELETE CASCADE,
    product_id          INTEGER NOT NULL REFERENCES products(id) ON DELETE RESTRICT,
    product_name        VARCHAR(200) NOT NULL,
    quantity            INTEGER NOT NULL CHECK (quantity > 0),
    unit_price          INTEGER NOT NULL CHECK (unit_price >= 0),
    original_price      INTEGER NOT NULL DEFAULT 0,
    discount            INTEGER NOT NULL DEFAULT 0,
    pricing_rule_id     INTEGER REFERENCES pricing_rules(id) ON DELETE SET NULL,
    pricing_rule_name   VARCHAR(200),
    pricing_rule_type   VARCHAR(50),
    pricing_type        VARCHAR(50),
    cost                INTEGER NOT NULL DEFAULT 0,
    tax_class_id        INTEGER REFERENCES tax_classes(id) ON DELETE SET NULL,
    tax_rate            NUMERIC(5,2),
    snapshot_created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    subtotal            INTEGER NOT NULL DEFAULT 0,
    dpp_amount          INTEGER NOT NULL DEFAULT 0,
    tax_amount          INTEGER NOT NULL DEFAULT 0,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT chk_cart_item_subtotal CHECK (subtotal = quantity * unit_price)
);

CREATE INDEX IF NOT EXISTS idx_cart_items_session ON cart_items(cart_session_id);

-- sale_items: add snapshot columns for full price history trace
ALTER TABLE sale_items
    ADD COLUMN IF NOT EXISTS cost                INTEGER NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS tax_class_id        INTEGER REFERENCES tax_classes(id) ON DELETE SET NULL,
    ADD COLUMN IF NOT EXISTS tax_rate            NUMERIC(5,2),
    ADD COLUMN IF NOT EXISTS snapshot_created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    ADD COLUMN IF NOT EXISTS product_name        VARCHAR(200);

-- Backfill product_name for historical sale items in small batches to avoid long locks.
DO $$
DECLARE
    batch_size INTEGER := 10000;
    last_id    INTEGER := 0;
    rows_updated INTEGER;
BEGIN
    LOOP
        UPDATE sale_items si
        SET product_name = p.name
        FROM products p
        WHERE si.product_id = p.id
          AND si.product_name IS NULL
          AND si.id > last_id
        ORDER BY si.id
        LIMIT batch_size
        RETURNING si.id INTO last_id;

        GET DIAGNOSTICS rows_updated = ROW_COUNT;
        IF rows_updated = 0 THEN
            EXIT;
        END IF;
        COMMIT;
        PERFORM pg_sleep(0.05);
    END LOOP;
END $$;

INSERT INTO schema_migrations (filename) VALUES ('010_sale_price_snapshot.sql');

COMMIT;
