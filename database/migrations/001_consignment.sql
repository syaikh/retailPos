-- Migration: 001_consignment.sql
-- Description: Konsinyasi Supplier feature (PRD-Konsinyasi-Supplier v7.1).
-- Adds supplier consignment flag, arrangements, terms, receipts, consignment
-- stock ownership, pending returns, returns, sale-item settlement tracking,
-- full settlements, and payout records. Idempotent (IF NOT EXISTS /
-- ON CONFLICT DO NOTHING), applied after 000_squash.sql.
-- Deployment ordering: apply BEFORE deploying the binary that references the
-- consignment_* tables/sequences and validates consignment.* permission codes.

BEGIN;

-- ============================================================
-- Standalone sequences (document numbers: CR-, RT-, CS-, CP-)
-- ============================================================
CREATE SEQUENCE IF NOT EXISTS consignment_receipt_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

CREATE SEQUENCE IF NOT EXISTS consignment_return_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

CREATE SEQUENCE IF NOT EXISTS consignment_settlement_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

CREATE SEQUENCE IF NOT EXISTS consignment_payout_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

-- ============================================================
-- Supplier consignment flag (BR-01)
-- ============================================================
ALTER TABLE suppliers ADD COLUMN IF NOT EXISTS is_consignment boolean DEFAULT false;

-- ============================================================
-- Tables
-- ============================================================

-- Consignment arrangement: one active per supplier+store (BR-47). The partial
-- unique index enforces the single-active-invariant while allowing a new active
-- arrangement after a previous one Ended.
CREATE TABLE IF NOT EXISTS consignment_arrangements (
    id SERIAL,
    supplier_id integer NOT NULL,
    store_id integer NOT NULL,
    status character varying(20) DEFAULT 'active'::character varying NOT NULL,
    last_visit_at timestamp with time zone,
    ended_at timestamp with time zone,
    created_by integer NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT chk_consignment_arrangement_status CHECK (((status)::text = ANY ((ARRAY['active'::character varying, 'ended'::character varying])::text[]))),
    CONSTRAINT consignment_arrangements_pkey PRIMARY KEY (id),
    CONSTRAINT consignment_arrangements_supplier_id_fkey FOREIGN KEY (supplier_id) REFERENCES suppliers(id),
    CONSTRAINT consignment_arrangements_store_id_fkey FOREIGN KEY (store_id) REFERENCES stores(id),
    CONSTRAINT consignment_arrangements_created_by_fkey FOREIGN KEY (created_by) REFERENCES users(id)
);

CREATE UNIQUE INDEX IF NOT EXISTS uq_consignment_arrangement_active
    ON consignment_arrangements (supplier_id, store_id) WHERE (status = 'active'::text);

CREATE INDEX IF NOT EXISTS idx_consignment_arrangements_supplier ON consignment_arrangements (supplier_id);
CREATE INDEX IF NOT EXISTS idx_consignment_arrangements_status ON consignment_arrangements (status);

-- Consignment terms: agreed price + exactly one store-share type (BR-14/AC-C09).
-- One current term per arrangement+product; the store share value is the fixed
-- amount (IDR) or the percentage (e.g. 20 for 20%).
CREATE TABLE IF NOT EXISTS consignment_terms (
    id SERIAL,
    arrangement_id integer NOT NULL,
    product_id integer NOT NULL,
    price integer NOT NULL,
    store_share_type character varying(20) NOT NULL,
    store_share_value numeric(18,4) NOT NULL,
    effective_from timestamp with time zone DEFAULT now() NOT NULL,
    created_by integer NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT chk_consignment_term_share_type CHECK (((store_share_type)::text = ANY ((ARRAY['percentage'::character varying, 'fixed_amount'::character varying])::text[]))),
    CONSTRAINT chk_consignment_term_share_value CHECK ((store_share_value > (0)::numeric)),
    CONSTRAINT chk_consignment_term_price CHECK ((price >= 0)),
    CONSTRAINT consignment_terms_pkey PRIMARY KEY (id),
    CONSTRAINT uq_consignment_terms_arrangement_product UNIQUE (arrangement_id, product_id),
    CONSTRAINT consignment_terms_arrangement_id_fkey FOREIGN KEY (arrangement_id) REFERENCES consignment_arrangements(id) ON DELETE CASCADE,
    CONSTRAINT consignment_terms_product_id_fkey FOREIGN KEY (product_id) REFERENCES products(id),
    CONSTRAINT consignment_terms_created_by_fkey FOREIGN KEY (created_by) REFERENCES users(id)
);

CREATE INDEX IF NOT EXISTS idx_consignment_terms_product ON consignment_terms (product_id);

-- Consignment receipt: records only ACCEPTED quantities after inspection
-- (BR-07/BR-10); rejected goods are never recorded (BR-08).
CREATE TABLE IF NOT EXISTS consignment_receipts (
    id SERIAL,
    receipt_number character varying(30) NOT NULL,
    supplier_id integer NOT NULL,
    store_id integer NOT NULL,
    arrangement_id integer NOT NULL,
    received_by integer NOT NULL,
    received_at timestamp with time zone DEFAULT now() NOT NULL,
    notes text,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT consignment_receipts_pkey PRIMARY KEY (id),
    CONSTRAINT consignment_receipts_receipt_number_key UNIQUE (receipt_number),
    CONSTRAINT consignment_receipts_supplier_id_fkey FOREIGN KEY (supplier_id) REFERENCES suppliers(id),
    CONSTRAINT consignment_receipts_store_id_fkey FOREIGN KEY (store_id) REFERENCES stores(id),
    CONSTRAINT consignment_receipts_arrangement_id_fkey FOREIGN KEY (arrangement_id) REFERENCES consignment_arrangements(id),
    CONSTRAINT consignment_receipts_received_by_fkey FOREIGN KEY (received_by) REFERENCES users(id)
);

CREATE TABLE IF NOT EXISTS consignment_receipt_items (
    id SERIAL,
    consignment_receipt_id integer NOT NULL,
    product_id integer NOT NULL,
    accepted_qty integer NOT NULL,
    price integer NOT NULL,
    store_share_type character varying(20) NOT NULL,
    store_share_value numeric(18,4) NOT NULL,
    notes text,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT chk_consignment_receipt_item_accepted CHECK ((accepted_qty > 0)),
    CONSTRAINT chk_consignment_receipt_item_share_type CHECK (((store_share_type)::text = ANY ((ARRAY['percentage'::character varying, 'fixed_amount'::character varying])::text[]))),
    CONSTRAINT consignment_receipt_items_pkey PRIMARY KEY (id),
    CONSTRAINT consignment_receipt_items_receipt_id_fkey FOREIGN KEY (consignment_receipt_id) REFERENCES consignment_receipts(id) ON DELETE CASCADE,
    CONSTRAINT consignment_receipt_items_product_id_fkey FOREIGN KEY (product_id) REFERENCES products(id)
);

-- Consignment stock ownership: one active owner per SKU (BR-03/AC-C35).
-- UNIQUE (product_id) enforces the single-owner invariant at the DB level.
-- Pending Return still counts as ownership (BR-05b/AC-C37).
CREATE TABLE IF NOT EXISTS consignment_stock (
    id SERIAL,
    product_id integer NOT NULL,
    supplier_id integer NOT NULL,
    arrangement_id integer NOT NULL,
    store_id integer NOT NULL,
    available_qty integer DEFAULT 0 NOT NULL,
    pending_return_qty integer DEFAULT 0 NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT chk_consignment_stock_available CHECK ((available_qty >= 0)),
    CONSTRAINT chk_consignment_stock_pending_return CHECK ((pending_return_qty >= 0)),
    CONSTRAINT consignment_stock_pkey PRIMARY KEY (id),
    CONSTRAINT uq_consignment_stock_product UNIQUE (product_id),
    CONSTRAINT consignment_stock_supplier_id_fkey FOREIGN KEY (supplier_id) REFERENCES suppliers(id),
    CONSTRAINT consignment_stock_arrangement_id_fkey FOREIGN KEY (arrangement_id) REFERENCES consignment_arrangements(id),
    CONSTRAINT consignment_stock_store_id_fkey FOREIGN KEY (store_id) REFERENCES stores(id),
    CONSTRAINT consignment_stock_product_id_fkey FOREIGN KEY (product_id) REFERENCES products(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_consignment_stock_supplier ON consignment_stock (supplier_id);

-- Pending Return: simple internal record (BR-29) for goods pulled off the
-- display and not yet handed back to the supplier.
CREATE TABLE IF NOT EXISTS consignment_pending_returns (
    id SERIAL,
    supplier_id integer NOT NULL,
    product_id integer NOT NULL,
    arrangement_id integer NOT NULL,
    store_id integer NOT NULL,
    qty integer NOT NULL,
    reason character varying(30) NOT NULL,
    notes text,
    status character varying(20) DEFAULT 'open'::character varying NOT NULL,
    returned_at timestamp with time zone,
    created_by integer NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT chk_consignment_pending_return_qty CHECK ((qty > 0)),
    CONSTRAINT chk_consignment_pending_return_reason CHECK (((reason)::text = ANY ((ARRAY['damaged'::character varying, 'expired'::character varying, 'customer_return'::character varying, 'other'::character varying])::text[]))),
    CONSTRAINT chk_consignment_pending_return_status CHECK (((status)::text = ANY ((ARRAY['open'::character varying, 'returned'::character varying])::text[]))),
    CONSTRAINT consignment_pending_returns_pkey PRIMARY KEY (id),
    CONSTRAINT consignment_pending_returns_supplier_id_fkey FOREIGN KEY (supplier_id) REFERENCES suppliers(id),
    CONSTRAINT consignment_pending_returns_product_id_fkey FOREIGN KEY (product_id) REFERENCES products(id),
    CONSTRAINT consignment_pending_returns_arrangement_id_fkey FOREIGN KEY (arrangement_id) REFERENCES consignment_arrangements(id),
    CONSTRAINT consignment_pending_returns_store_id_fkey FOREIGN KEY (store_id) REFERENCES stores(id),
    CONSTRAINT consignment_pending_returns_created_by_fkey FOREIGN KEY (created_by) REFERENCES users(id)
);

CREATE INDEX IF NOT EXISTS idx_consignment_pending_returns_open ON consignment_pending_returns (supplier_id, status);

-- Formal Consignment Return: goods actually handed back to the supplier
-- (BR-31). Does NOT generate settlement (BR-32).
CREATE TABLE IF NOT EXISTS consignment_returns (
    id SERIAL,
    return_number character varying(30) NOT NULL,
    supplier_id integer NOT NULL,
    store_id integer NOT NULL,
    arrangement_id integer NOT NULL,
    returned_by integer NOT NULL,
    returned_at timestamp with time zone DEFAULT now() NOT NULL,
    notes text,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT consignment_returns_pkey PRIMARY KEY (id),
    CONSTRAINT consignment_returns_return_number_key UNIQUE (return_number),
    CONSTRAINT consignment_returns_supplier_id_fkey FOREIGN KEY (supplier_id) REFERENCES suppliers(id),
    CONSTRAINT consignment_returns_store_id_fkey FOREIGN KEY (store_id) REFERENCES stores(id),
    CONSTRAINT consignment_returns_arrangement_id_fkey FOREIGN KEY (arrangement_id) REFERENCES consignment_arrangements(id),
    CONSTRAINT consignment_returns_returned_by_fkey FOREIGN KEY (returned_by) REFERENCES users(id)
);

CREATE TABLE IF NOT EXISTS consignment_return_items (
    id SERIAL,
    consignment_return_id integer NOT NULL,
    product_id integer NOT NULL,
    qty integer NOT NULL,
    reason character varying(30) NOT NULL,
    pending_return_id integer,
    notes text,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT chk_consignment_return_item_qty CHECK ((qty > 0)),
    CONSTRAINT consignment_return_items_pkey PRIMARY KEY (id),
    CONSTRAINT consignment_return_items_return_id_fkey FOREIGN KEY (consignment_return_id) REFERENCES consignment_returns(id) ON DELETE CASCADE,
    CONSTRAINT consignment_return_items_product_id_fkey FOREIGN KEY (product_id) REFERENCES products(id),
    CONSTRAINT consignment_return_items_pending_return_id_fkey FOREIGN KEY (pending_return_id) REFERENCES consignment_pending_returns(id) ON DELETE SET NULL
);

-- Consignment sale items: recorded at checkout inside the sale Unit of Work.
-- Store share is snapshotted at sale time (BR-19/BR-43/AC-C11/AC-C12);
-- unsettled = settlement_id IS NULL (BR-24/AC-C18).
CREATE TABLE IF NOT EXISTS consignment_sale_items (
    id SERIAL,
    sale_id integer NOT NULL,
    invoice_number character varying(50) NOT NULL,
    product_id integer NOT NULL,
    supplier_id integer NOT NULL,
    arrangement_id integer NOT NULL,
    store_id integer NOT NULL,
    quantity integer NOT NULL,
    unit_price integer NOT NULL,
    subtotal integer NOT NULL,
    store_share_type character varying(20) NOT NULL,
    store_share_value numeric(18,4) NOT NULL,
    settlement_id integer,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT chk_consignment_sale_item_quantity CHECK ((quantity > 0)),
    CONSTRAINT chk_consignment_sale_item_share_type CHECK (((store_share_type)::text = ANY ((ARRAY['percentage'::character varying, 'fixed_amount'::character varying])::text[]))),
    CONSTRAINT consignment_sale_items_pkey PRIMARY KEY (id),
    CONSTRAINT consignment_sale_items_sale_id_fkey FOREIGN KEY (sale_id) REFERENCES sales(id) ON DELETE CASCADE,
    CONSTRAINT consignment_sale_items_product_id_fkey FOREIGN KEY (product_id) REFERENCES products(id),
    CONSTRAINT consignment_sale_items_supplier_id_fkey FOREIGN KEY (supplier_id) REFERENCES suppliers(id),
    CONSTRAINT consignment_sale_items_arrangement_id_fkey FOREIGN KEY (arrangement_id) REFERENCES consignment_arrangements(id),
    CONSTRAINT consignment_sale_items_store_id_fkey FOREIGN KEY (store_id) REFERENCES stores(id)
);

CREATE INDEX IF NOT EXISTS idx_consignment_sale_items_unsettled ON consignment_sale_items (supplier_id, settlement_id);

-- Full settlement (BR-41/AC-C27/AC-C28): covers ALL unsettled sales of a
-- supplier at once. Partial settlement is never allowed.
CREATE TABLE IF NOT EXISTS consignment_settlements (
    id SERIAL,
    settlement_number character varying(30) NOT NULL,
    supplier_id integer NOT NULL,
    store_id integer NOT NULL,
    total_sale_value integer DEFAULT 0 NOT NULL,
    total_store_share integer DEFAULT 0 NOT NULL,
    total_payable integer DEFAULT 0 NOT NULL,
    status character varying(20) DEFAULT 'pending_payment'::character varying NOT NULL,
    created_by integer NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    paid_at timestamp with time zone,
    CONSTRAINT chk_consignment_settlement_status CHECK (((status)::text = ANY ((ARRAY['pending_payment'::character varying, 'paid'::character varying])::text[]))),
    CONSTRAINT consignment_settlements_pkey PRIMARY KEY (id),
    CONSTRAINT consignment_settlements_settlement_number_key UNIQUE (settlement_number),
    CONSTRAINT consignment_settlements_supplier_id_fkey FOREIGN KEY (supplier_id) REFERENCES suppliers(id),
    CONSTRAINT consignment_settlements_store_id_fkey FOREIGN KEY (store_id) REFERENCES stores(id),
    CONSTRAINT consignment_settlements_created_by_fkey FOREIGN KEY (created_by) REFERENCES users(id)
);

CREATE TABLE IF NOT EXISTS consignment_settlement_items (
    id SERIAL,
    consignment_settlement_id integer NOT NULL,
    consignment_sale_item_id integer NOT NULL,
    quantity integer NOT NULL,
    unit_price integer NOT NULL,
    subtotal integer NOT NULL,
    store_share integer NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT consignment_settlement_items_pkey PRIMARY KEY (id),
    CONSTRAINT consignment_settlement_items_settlement_id_fkey FOREIGN KEY (consignment_settlement_id) REFERENCES consignment_settlements(id) ON DELETE CASCADE,
    CONSTRAINT consignment_settlement_items_sale_item_id_fkey FOREIGN KEY (consignment_sale_item_id) REFERENCES consignment_sale_items(id)
);

CREATE INDEX IF NOT EXISTS idx_consignment_settlement_items_settlement ON consignment_settlement_items (consignment_settlement_id);

-- Payout: money-out to supplier, reusing the existing payment_methods registry
-- (BR-44/AC-C30). Decoupled from sale payments.
CREATE TABLE IF NOT EXISTS consignment_payouts (
    id SERIAL,
    payout_number character varying(30) NOT NULL,
    settlement_id integer NOT NULL,
    payment_method_id integer NOT NULL,
    amount integer NOT NULL,
    reference_number character varying(100),
    paid_by integer NOT NULL,
    paid_at timestamp with time zone DEFAULT now() NOT NULL,
    notes text,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT chk_consignment_payout_amount CHECK ((amount > 0)),
    CONSTRAINT consignment_payouts_pkey PRIMARY KEY (id),
    CONSTRAINT consignment_payouts_payout_number_key UNIQUE (payout_number),
    CONSTRAINT consignment_payouts_settlement_id_fkey FOREIGN KEY (settlement_id) REFERENCES consignment_settlements(id),
    CONSTRAINT consignment_payouts_payment_method_id_fkey FOREIGN KEY (payment_method_id) REFERENCES payment_methods(id),
    CONSTRAINT consignment_payouts_paid_by_fkey FOREIGN KEY (paid_by) REFERENCES users(id)
);

-- ============================================================
-- Permissions & role grants
-- ============================================================
INSERT INTO permissions (code, name, description) VALUES
    ('consignment.view', 'Lihat Konsinyasi', 'Bisa melihat arrangement, stock, dan settlement konsinyasi'),
    ('consignment.create', 'Buat Transaksi Konsinyasi', 'Bisa membuat arrangement, receipt, pending return, dan return'),
    ('consignment.update', 'Ubah Terms Konsinyasi', 'Bisa mengubah harga dan hak/potongan toko pada arrangement konsinyasi'),
    ('consignment.settle', 'Settlement Konsinyasi', 'Bisa membuat settlement konsinyasi'),
    ('consignment.pay', 'Pembayaran Konsinyasi', 'Bisa mencatat pembayaran kepada supplier konsinyasi')
ON CONFLICT (code) DO NOTHING;

-- Superadmin
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM roles r, permissions p WHERE r.name = 'superadmin'
    AND p.code IN (
        'consignment.view',
        'consignment.create',
        'consignment.update',
        'consignment.settle',
        'consignment.pay'
    )
ON CONFLICT DO NOTHING;

-- Admin
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM roles r, permissions p WHERE r.name = 'admin'
    AND p.code IN (
        'consignment.view',
        'consignment.create',
        'consignment.update',
        'consignment.settle',
        'consignment.pay'
    )
ON CONFLICT DO NOTHING;

-- Manager
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM roles r, permissions p WHERE r.name = 'manager'
    AND p.code IN (
        'consignment.view',
        'consignment.create',
        'consignment.update',
        'consignment.settle'
    )
ON CONFLICT DO NOTHING;

-- ============================================================
-- Migration registration (idempotent)
-- ============================================================
INSERT INTO schema_migrations (filename) VALUES ('001_consignment.sql')
ON CONFLICT (filename) DO NOTHING;

COMMIT;