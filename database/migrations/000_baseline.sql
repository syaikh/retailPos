-- ============================================================================
-- Migration: 000_baseline.sql
-- Description: Version 1 baseline — the complete schema, functions, indexes,
--              constraints and reference data produced by the full legacy
--              migration chain (000_squash.sql + 001..053), squashed into a
--              single idempotent file.
--
-- Generated from a pg_dump of a fully-migrated reference database and
-- normalised so that it replays safely on an empty database, on a database
-- that already went through the legacy chain, and on every re-run:
--
--   * every DDL statement is CREATE ... IF NOT EXISTS / DROP ... IF EXISTS
--     or guarded by a DO block that checks pg_catalog first
--   * every reference-data row is INSERT ... ON CONFLICT DO NOTHING
--   * materialised views are created WITH DATA (empty until refreshed)
--   * no frozen timestamps are seeded: columns with DEFAULT now() are left
--     to their defaults
--   * users.password_hash is regenerated as crypt('admin123', gen_salt(...))
--     so each install gets its own salt and no literal credential hash is
--     committed to the repository
--
-- The 32 migrations this replaces are archived in
-- database/migrations/archive/pre-squash-migrations.tar.gz.
--
-- Runners: deploy/podman-deploy.sh, .github/workflows/ci.yml,
-- .github/workflows/e2e.yml and internal/shared/testdb.go all iterate
-- database/migrations/*.sql, so this file must stay fully re-runnable.
-- testdb.go additionally requires len(schema_migrations) == len(files), which
-- is why the last step below clears the ledger rows of the replaced
-- migrations and registers this file.
--
-- The whole file runs in one transaction: a failure leaves the database
-- exactly as it was, and the ledger is only rewritten once the schema and
-- reference data are in place.
-- ============================================================================

BEGIN;

-- ============================================================================
-- Extensions
-- ============================================================================

CREATE EXTENSION IF NOT EXISTS pg_trgm WITH SCHEMA public;

COMMENT ON EXTENSION pg_trgm IS 'text similarity measurement and index searching based on trigrams';

CREATE EXTENSION IF NOT EXISTS pgcrypto WITH SCHEMA public;

COMMENT ON EXTENSION pgcrypto IS 'cryptographic functions';

-- ============================================================================
-- Functions
-- ============================================================================

CREATE OR REPLACE FUNCTION public.prevent_consignment_receipt_edit_modification() RETURNS trigger
    LANGUAGE plpgsql
    AS $$
BEGIN
    RAISE EXCEPTION 'consignment_receipt_edits is append-only: modifications are not permitted';
END;
$$;

CREATE OR REPLACE FUNCTION public.products_search_vector_update() RETURNS trigger
    LANGUAGE plpgsql
    AS $$
BEGIN
    NEW.search_vector :=
        setweight(to_tsvector('english', coalesce(NEW.name, '')), 'A') ||
        setweight(to_tsvector('english', coalesce(NEW.sku, '')), 'B') ||
        setweight(to_tsvector('english', coalesce(NEW.barcode, '')), 'C');
    RETURN NEW;
END;
$$;

CREATE OR REPLACE FUNCTION public.refresh_sales_mv() RETURNS void
    LANGUAGE plpgsql
    AS $$
BEGIN
    REFRESH MATERIALIZED VIEW CONCURRENTLY mv_daily_sales;
    REFRESH MATERIALIZED VIEW CONCURRENTLY mv_hourly_sales;
    REFRESH MATERIALIZED VIEW CONCURRENTLY mv_dashboard_totals;
END;
$$;

CREATE OR REPLACE FUNCTION public.reject_audit_log_modification() RETURNS trigger
    LANGUAGE plpgsql
    AS $$
BEGIN
  -- FK cascades fire at depth >= 1; allow them through.
  IF pg_trigger_depth() > 1 THEN
    RETURN NULL;
  END IF;
  IF current_setting('app.allow_audit_mod', true) = 'on' THEN
    RETURN NULL;
  END IF;
  RAISE EXCEPTION 'audit_logs is append-only: modifications are not permitted';
END;
$$;

-- ============================================================================
-- Tables, sequences, views and materialized views
-- ============================================================================

CREATE TABLE IF NOT EXISTS public.app_settings (
    key character varying(100) NOT NULL,
    value text DEFAULT ''::text NOT NULL,
    updated_at timestamp with time zone DEFAULT now()
);

CREATE TABLE IF NOT EXISTS public.audit_logs (
    id integer NOT NULL,
    user_id integer,
    role character varying(50),
    action character varying(100) NOT NULL,
    entity_type character varying(100),
    entity_id integer,
    old_values jsonb,
    new_values jsonb,
    description text,
    ip_address inet,
    user_agent text,
    created_at timestamp with time zone DEFAULT now(),
    store_id integer,
    correlation_id text
);

COMMENT ON COLUMN public.audit_logs.correlation_id IS 'Request correlation ID (X-Request-ID) for tracing related audit events within a single request.';

CREATE SEQUENCE IF NOT EXISTS public.audit_logs_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

ALTER SEQUENCE public.audit_logs_id_seq OWNED BY public.audit_logs.id;

CREATE TABLE IF NOT EXISTS public.brands (
    id integer NOT NULL,
    name character varying(100) NOT NULL,
    description text,
    is_active boolean DEFAULT true,
    created_at timestamp with time zone DEFAULT now(),
    updated_at timestamp with time zone DEFAULT now()
);

CREATE SEQUENCE IF NOT EXISTS public.brands_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

ALTER SEQUENCE public.brands_id_seq OWNED BY public.brands.id;

CREATE TABLE IF NOT EXISTS public.cart_items (
    id integer NOT NULL,
    cart_session_id integer NOT NULL,
    product_id integer NOT NULL,
    product_name character varying(200) NOT NULL,
    quantity integer NOT NULL,
    unit_price integer NOT NULL,
    original_price integer DEFAULT 0 NOT NULL,
    discount integer DEFAULT 0 NOT NULL,
    pricing_rule_id integer,
    pricing_rule_name character varying(200),
    pricing_rule_type character varying(50),
    pricing_type character varying(50),
    cost integer DEFAULT 0 NOT NULL,
    tax_class_id integer,
    tax_rate numeric(5,2),
    snapshot_created_at timestamp with time zone DEFAULT now() NOT NULL,
    subtotal integer DEFAULT 0 NOT NULL,
    dpp_amount integer DEFAULT 0 NOT NULL,
    tax_amount integer DEFAULT 0 NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT cart_items_quantity_check CHECK ((quantity > 0)),
    CONSTRAINT cart_items_unit_price_check CHECK ((unit_price >= 0)),
    CONSTRAINT chk_cart_item_subtotal CHECK ((subtotal = (quantity * unit_price)))
);

CREATE SEQUENCE IF NOT EXISTS public.cart_items_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

ALTER SEQUENCE public.cart_items_id_seq OWNED BY public.cart_items.id;

CREATE TABLE IF NOT EXISTS public.cart_sessions (
    id integer NOT NULL,
    cashier_id integer NOT NULL,
    store_id integer,
    shift_id integer,
    customer_id integer,
    status character varying(20) DEFAULT 'open'::character varying NOT NULL,
    subtotal integer DEFAULT 0 NOT NULL,
    discount integer DEFAULT 0 NOT NULL,
    tax integer DEFAULT 0 NOT NULL,
    total_amount integer DEFAULT 0 NOT NULL,
    expired_at timestamp with time zone,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT cart_sessions_status_check CHECK (((status)::text = ANY (ARRAY[('open'::character varying)::text, ('held'::character varying)::text, ('checked_out'::character varying)::text, ('cancelled'::character varying)::text, ('expired'::character varying)::text])))
);

CREATE SEQUENCE IF NOT EXISTS public.cart_sessions_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

ALTER SEQUENCE public.cart_sessions_id_seq OWNED BY public.cart_sessions.id;

CREATE TABLE IF NOT EXISTS public.cash_movements (
    id integer NOT NULL,
    shift_id integer NOT NULL,
    user_id integer NOT NULL,
    type character varying(20) NOT NULL,
    amount integer NOT NULL,
    description text,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT cash_movements_amount_check CHECK ((amount > 0)),
    CONSTRAINT cash_movements_type_check CHECK (type IN ('cash_drop', 'paid_in', 'paid_out'))
);

CREATE SEQUENCE IF NOT EXISTS public.cash_movements_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

ALTER SEQUENCE public.cash_movements_id_seq OWNED BY public.cash_movements.id;

CREATE TABLE IF NOT EXISTS public.categories (
    id integer NOT NULL,
    name character varying(100) NOT NULL,
    description text,
    parent_id integer,
    slug character varying(120),
    is_active boolean DEFAULT true,
    created_at timestamp with time zone DEFAULT now(),
    updated_at timestamp with time zone DEFAULT now()
);

CREATE SEQUENCE IF NOT EXISTS public.categories_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

ALTER SEQUENCE public.categories_id_seq OWNED BY public.categories.id;

CREATE TABLE IF NOT EXISTS public.consignment_arrangements (
    id integer NOT NULL,
    supplier_id integer NOT NULL,
    store_id integer NOT NULL,
    status character varying(20) DEFAULT 'active'::character varying NOT NULL,
    last_visit_at timestamp with time zone,
    ended_at timestamp with time zone,
    created_by integer NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT chk_consignment_arrangement_status CHECK (((status)::text = ANY (ARRAY[('active'::character varying)::text, ('ended'::character varying)::text])))
);

CREATE SEQUENCE IF NOT EXISTS public.consignment_arrangements_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

ALTER SEQUENCE public.consignment_arrangements_id_seq OWNED BY public.consignment_arrangements.id;

CREATE SEQUENCE IF NOT EXISTS public.consignment_payout_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

CREATE TABLE IF NOT EXISTS public.consignment_payouts (
    id integer NOT NULL,
    payout_number character varying(30) NOT NULL,
    settlement_id integer NOT NULL,
    payment_method_id integer NOT NULL,
    amount integer NOT NULL,
    reference_number character varying(100),
    paid_by integer NOT NULL,
    paid_at timestamp with time zone DEFAULT now() NOT NULL,
    notes text,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT chk_consignment_payout_amount CHECK ((amount > 0))
);

CREATE SEQUENCE IF NOT EXISTS public.consignment_payouts_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

ALTER SEQUENCE public.consignment_payouts_id_seq OWNED BY public.consignment_payouts.id;

CREATE TABLE IF NOT EXISTS public.consignment_pending_returns (
    id integer NOT NULL,
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
    CONSTRAINT chk_consignment_pending_return_reason CHECK (((reason)::text = ANY (ARRAY[('damaged'::character varying)::text, ('expired'::character varying)::text, ('customer_return'::character varying)::text, ('termination'::character varying)::text, ('other'::character varying)::text]))),
    CONSTRAINT chk_consignment_pending_return_status CHECK (((status)::text = ANY (ARRAY[('open'::character varying)::text, ('returned'::character varying)::text])))
);

CREATE SEQUENCE IF NOT EXISTS public.consignment_pending_returns_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

ALTER SEQUENCE public.consignment_pending_returns_id_seq OWNED BY public.consignment_pending_returns.id;

CREATE TABLE IF NOT EXISTS public.consignment_receipt_edits (
    id integer NOT NULL,
    receipt_id integer NOT NULL,
    edited_by integer NOT NULL,
    edited_at timestamp with time zone DEFAULT now() NOT NULL,
    field text NOT NULL,
    old_value text NOT NULL,
    new_value text NOT NULL,
    reason text NOT NULL,
    ip_address inet,
    created_at timestamp with time zone DEFAULT now() NOT NULL
);

COMMENT ON TABLE public.consignment_receipt_edits IS 'Immutable audit trail for consignment receipt edits';

CREATE SEQUENCE IF NOT EXISTS public.consignment_receipt_edits_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

ALTER SEQUENCE public.consignment_receipt_edits_id_seq OWNED BY public.consignment_receipt_edits.id;

CREATE TABLE IF NOT EXISTS public.consignment_receipt_items (
    id integer NOT NULL,
    consignment_receipt_id integer NOT NULL,
    product_id integer NOT NULL,
    accepted_qty integer NOT NULL,
    price integer NOT NULL,
    store_share_type character varying(20) NOT NULL,
    store_share_value numeric(18,4) NOT NULL,
    notes text,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT chk_consignment_receipt_item_accepted CHECK ((accepted_qty > 0)),
    CONSTRAINT chk_consignment_receipt_item_share_type CHECK (((store_share_type)::text = ANY (ARRAY[('percentage'::character varying)::text, ('fixed_amount'::character varying)::text])))
);

CREATE SEQUENCE IF NOT EXISTS public.consignment_receipt_items_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

ALTER SEQUENCE public.consignment_receipt_items_id_seq OWNED BY public.consignment_receipt_items.id;

CREATE SEQUENCE IF NOT EXISTS public.consignment_receipt_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

CREATE TABLE IF NOT EXISTS public.consignment_receipts (
    id integer NOT NULL,
    receipt_number character varying(30) NOT NULL,
    supplier_id integer NOT NULL,
    store_id integer NOT NULL,
    arrangement_id integer NOT NULL,
    received_by integer NOT NULL,
    received_at timestamp with time zone DEFAULT now() NOT NULL,
    notes text,
    created_at timestamp with time zone DEFAULT now() NOT NULL
);

CREATE SEQUENCE IF NOT EXISTS public.consignment_receipts_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

ALTER SEQUENCE public.consignment_receipts_id_seq OWNED BY public.consignment_receipts.id;

CREATE TABLE IF NOT EXISTS public.consignment_return_items (
    id integer NOT NULL,
    consignment_return_id integer NOT NULL,
    product_id integer NOT NULL,
    qty integer NOT NULL,
    reason character varying(30) NOT NULL,
    pending_return_id integer,
    notes text,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT chk_consignment_return_item_qty CHECK ((qty > 0))
);

CREATE SEQUENCE IF NOT EXISTS public.consignment_return_items_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

ALTER SEQUENCE public.consignment_return_items_id_seq OWNED BY public.consignment_return_items.id;

CREATE SEQUENCE IF NOT EXISTS public.consignment_return_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

CREATE TABLE IF NOT EXISTS public.consignment_returns (
    id integer NOT NULL,
    return_number character varying(30) NOT NULL,
    supplier_id integer NOT NULL,
    store_id integer NOT NULL,
    arrangement_id integer NOT NULL,
    returned_by integer NOT NULL,
    returned_at timestamp with time zone DEFAULT now() NOT NULL,
    notes text,
    created_at timestamp with time zone DEFAULT now() NOT NULL
);

CREATE SEQUENCE IF NOT EXISTS public.consignment_returns_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

ALTER SEQUENCE public.consignment_returns_id_seq OWNED BY public.consignment_returns.id;

CREATE TABLE IF NOT EXISTS public.consignment_sale_items (
    id integer NOT NULL,
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
    CONSTRAINT chk_consignment_sale_item_share_type CHECK (((store_share_type)::text = ANY (ARRAY[('percentage'::character varying)::text, ('fixed_amount'::character varying)::text])))
);

CREATE SEQUENCE IF NOT EXISTS public.consignment_sale_items_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

ALTER SEQUENCE public.consignment_sale_items_id_seq OWNED BY public.consignment_sale_items.id;

CREATE TABLE IF NOT EXISTS public.consignment_settlement_items (
    id integer NOT NULL,
    consignment_settlement_id integer NOT NULL,
    consignment_sale_item_id integer NOT NULL,
    quantity integer NOT NULL,
    unit_price integer NOT NULL,
    subtotal integer NOT NULL,
    store_share integer NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    product_id integer
);

CREATE SEQUENCE IF NOT EXISTS public.consignment_settlement_items_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

ALTER SEQUENCE public.consignment_settlement_items_id_seq OWNED BY public.consignment_settlement_items.id;

CREATE SEQUENCE IF NOT EXISTS public.consignment_settlement_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

CREATE TABLE IF NOT EXISTS public.consignment_settlements (
    id integer NOT NULL,
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
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT chk_consignment_settlement_status CHECK (((status)::text = ANY (ARRAY[('pending_payment'::character varying)::text, ('paid'::character varying)::text])))
);

CREATE SEQUENCE IF NOT EXISTS public.consignment_settlements_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

ALTER SEQUENCE public.consignment_settlements_id_seq OWNED BY public.consignment_settlements.id;

CREATE TABLE IF NOT EXISTS public.consignment_stock (
    id integer NOT NULL,
    product_id integer NOT NULL,
    supplier_id integer NOT NULL,
    arrangement_id integer NOT NULL,
    store_id integer NOT NULL,
    available_qty integer DEFAULT 0 NOT NULL,
    pending_return_qty integer DEFAULT 0 NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT chk_consignment_stock_available CHECK ((available_qty >= 0)),
    CONSTRAINT chk_consignment_stock_pending_return CHECK ((pending_return_qty >= 0))
);

CREATE SEQUENCE IF NOT EXISTS public.consignment_stock_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

ALTER SEQUENCE public.consignment_stock_id_seq OWNED BY public.consignment_stock.id;

CREATE TABLE IF NOT EXISTS public.consignment_terms (
    id integer NOT NULL,
    arrangement_id integer NOT NULL,
    product_id integer NOT NULL,
    price integer NOT NULL,
    store_share_type character varying(20) NOT NULL,
    store_share_value numeric(18,4) NOT NULL,
    effective_from timestamp with time zone DEFAULT now() NOT NULL,
    created_by integer NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT chk_consignment_term_price CHECK ((price >= 0)),
    CONSTRAINT chk_consignment_term_share_type CHECK (((store_share_type)::text = ANY (ARRAY[('percentage'::character varying)::text, ('fixed_amount'::character varying)::text]))),
    CONSTRAINT chk_consignment_term_share_value CHECK ((store_share_value > (0)::numeric))
);

CREATE SEQUENCE IF NOT EXISTS public.consignment_terms_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

ALTER SEQUENCE public.consignment_terms_id_seq OWNED BY public.consignment_terms.id;

CREATE TABLE IF NOT EXISTS public.customer_groups (
    id integer NOT NULL,
    name character varying(100) NOT NULL,
    description text,
    is_active boolean DEFAULT true NOT NULL,
    color character varying(7),
    created_at timestamp with time zone DEFAULT now(),
    updated_at timestamp with time zone DEFAULT now()
);

CREATE SEQUENCE IF NOT EXISTS public.customer_groups_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

ALTER SEQUENCE public.customer_groups_id_seq OWNED BY public.customer_groups.id;

CREATE TABLE IF NOT EXISTS public.customers (
    id integer NOT NULL,
    name character varying(200) NOT NULL,
    phone character varying(20) NOT NULL,
    email character varying(100) NOT NULL,
    address text,
    tax_id character varying(50),
    loyalty_points integer DEFAULT 0 NOT NULL,
    total_spent integer DEFAULT 0 NOT NULL,
    last_purchase_at timestamp with time zone,
    note text,
    is_active boolean DEFAULT true,
    is_walk_in boolean DEFAULT false,
    store_id integer DEFAULT 1 NOT NULL,
    customer_group_id integer,
    created_at timestamp with time zone DEFAULT now(),
    updated_at timestamp with time zone DEFAULT now()
);

CREATE SEQUENCE IF NOT EXISTS public.customers_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

ALTER SEQUENCE public.customers_id_seq OWNED BY public.customers.id;

CREATE TABLE IF NOT EXISTS public.dead_letter_events (
    id bigint NOT NULL,
    event_type text NOT NULL,
    payload jsonb,
    error text NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL
);

CREATE SEQUENCE IF NOT EXISTS public.dead_letter_events_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

ALTER SEQUENCE public.dead_letter_events_id_seq OWNED BY public.dead_letter_events.id;

CREATE SEQUENCE IF NOT EXISTS public.do_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

CREATE TABLE IF NOT EXISTS public.goods_receipt_items (
    id integer NOT NULL,
    goods_receipt_id integer NOT NULL,
    purchase_order_item_id integer NOT NULL,
    product_id integer NOT NULL,
    qty_good integer NOT NULL,
    qty_damaged integer DEFAULT 0 NOT NULL,
    unit_cost integer DEFAULT 0 NOT NULL,
    product_name character varying(255) DEFAULT ''::character varying NOT NULL,
    supplier_id integer,
    notes text,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT goods_receipt_items_qty_damaged_check CHECK ((qty_damaged >= 0)),
    CONSTRAINT goods_receipt_items_qty_good_check CHECK ((qty_good >= 0))
);

CREATE SEQUENCE IF NOT EXISTS public.goods_receipt_items_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

ALTER SEQUENCE public.goods_receipt_items_id_seq OWNED BY public.goods_receipt_items.id;

CREATE TABLE IF NOT EXISTS public.goods_receipts (
    id integer NOT NULL,
    gr_number character varying(30) NOT NULL,
    purchase_order_id integer NOT NULL,
    store_id integer NOT NULL,
    received_by integer NOT NULL,
    received_at timestamp with time zone DEFAULT now() NOT NULL,
    delivery_order_number character varying(100),
    shipping_method character varying(50),
    driver_name character varying(100),
    vehicle_plate_number character varying(20),
    notes text,
    created_at timestamp with time zone DEFAULT now() NOT NULL
);

CREATE SEQUENCE IF NOT EXISTS public.goods_receipts_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

ALTER SEQUENCE public.goods_receipts_id_seq OWNED BY public.goods_receipts.id;

CREATE SEQUENCE IF NOT EXISTS public.gr_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

CREATE SEQUENCE IF NOT EXISTS public.ia_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

CREATE TABLE IF NOT EXISTS public.import_errors (
    id bigint NOT NULL,
    import_job_id bigint NOT NULL,
    row_number integer NOT NULL,
    field character varying(100),
    value text,
    reason text NOT NULL,
    suggestion text,
    stage character varying(30) NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL
);

CREATE SEQUENCE IF NOT EXISTS public.import_errors_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

ALTER SEQUENCE public.import_errors_id_seq OWNED BY public.import_errors.id;

CREATE TABLE IF NOT EXISTS public.import_jobs (
    id bigint NOT NULL,
    module character varying(50) NOT NULL,
    schema_version character varying(20) NOT NULL,
    filename character varying(255) NOT NULL,
    status character varying(20) DEFAULT 'queued'::character varying NOT NULL,
    total_rows integer DEFAULT 0 NOT NULL,
    inserted integer DEFAULT 0 NOT NULL,
    updated integer DEFAULT 0 NOT NULL,
    skipped integer DEFAULT 0 NOT NULL,
    error_count integer DEFAULT 0 NOT NULL,
    progress_pct integer DEFAULT 0 NOT NULL,
    error_report_path text,
    started_at timestamp with time zone,
    completed_at timestamp with time zone,
    duration_ms integer,
    user_id integer NOT NULL,
    store_id integer,
    cancel_requested boolean DEFAULT false NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL
);

CREATE SEQUENCE IF NOT EXISTS public.import_jobs_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

ALTER SEQUENCE public.import_jobs_id_seq OWNED BY public.import_jobs.id;

CREATE TABLE IF NOT EXISTS public.import_rows (
    id bigint NOT NULL,
    import_job_id bigint NOT NULL,
    row_number integer NOT NULL,
    status character varying(20) NOT NULL,
    entity_id integer,
    old_values jsonb,
    new_values jsonb,
    changed_fields text[],
    created_at timestamp with time zone DEFAULT now() NOT NULL
);

CREATE SEQUENCE IF NOT EXISTS public.import_rows_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

ALTER SEQUENCE public.import_rows_id_seq OWNED BY public.import_rows.id;

CREATE TABLE IF NOT EXISTS public.import_snapshots (
    id bigint NOT NULL,
    import_job_id bigint NOT NULL,
    rows_data jsonb NOT NULL,
    schema_snapshot jsonb NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL
);

CREATE SEQUENCE IF NOT EXISTS public.import_snapshots_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

ALTER SEQUENCE public.import_snapshots_id_seq OWNED BY public.import_snapshots.id;

CREATE TABLE IF NOT EXISTS public.inventory_adjustment_items (
    id bigint NOT NULL,
    adjustment_id integer NOT NULL,
    product_id integer NOT NULL,
    warehouse_id integer,
    store_id integer,
    expected_qty numeric(18,4) DEFAULT 0 NOT NULL,
    physical_qty numeric(18,4) DEFAULT 0 NOT NULL,
    difference_qty numeric(18,4) DEFAULT 0 NOT NULL,
    adjustment_qty numeric(18,4) DEFAULT 0 NOT NULL,
    unit_cost numeric(18,4) DEFAULT 0 NOT NULL,
    line_total numeric(18,4) DEFAULT 0 NOT NULL,
    reason text,
    created_at timestamp with time zone DEFAULT now() NOT NULL
);

CREATE SEQUENCE IF NOT EXISTS public.inventory_adjustment_items_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

ALTER SEQUENCE public.inventory_adjustment_items_id_seq OWNED BY public.inventory_adjustment_items.id;

CREATE TABLE IF NOT EXISTS public.inventory_adjustments (
    id bigint NOT NULL,
    adjustment_number character varying(30) NOT NULL,
    session_id integer NOT NULL,
    status character varying(20) DEFAULT 'posted'::character varying NOT NULL,
    notes text,
    created_by integer,
    created_at timestamp with time zone DEFAULT now() NOT NULL
);

CREATE SEQUENCE IF NOT EXISTS public.inventory_adjustments_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

ALTER SEQUENCE public.inventory_adjustments_id_seq OWNED BY public.inventory_adjustments.id;

CREATE TABLE IF NOT EXISTS public.inventory_movements (
    id integer NOT NULL,
    product_id integer NOT NULL,
    quantity_change integer NOT NULL,
    type character varying(50) NOT NULL,
    reference_id integer,
    reference_table character varying(50),
    user_id integer,
    notes text,
    created_at timestamp with time zone DEFAULT now()
);

CREATE SEQUENCE IF NOT EXISTS public.inventory_movements_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

ALTER SEQUENCE public.inventory_movements_id_seq OWNED BY public.inventory_movements.id;

CREATE SEQUENCE IF NOT EXISTS public.invoice_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

CREATE TABLE IF NOT EXISTS public.sales (
    id integer NOT NULL,
    invoice_number character varying(50) NOT NULL,
    cashier_id integer NOT NULL,
    store_id integer,
    customer_id integer DEFAULT 1,
    shift_id integer,
    subtotal integer DEFAULT 0 NOT NULL,
    discount integer DEFAULT 0,
    tax integer DEFAULT 0,
    total_amount integer DEFAULT 0 NOT NULL,
    payment_method character varying(50) NOT NULL,
    status character varying(20) DEFAULT 'completed'::character varying,
    created_at timestamp with time zone DEFAULT now(),
    updated_at timestamp with time zone DEFAULT now(),
    hold_note text,
    change_due integer DEFAULT 0 NOT NULL
);

CREATE MATERIALIZED VIEW IF NOT EXISTS public.mv_daily_sales AS
 SELECT date((created_at AT TIME ZONE 'Asia/Jakarta'::text)) AS sale_date,
    store_id,
    count(*) AS transaction_count,
    count(DISTINCT cashier_id) AS active_cashiers,
    COALESCE(sum(total_amount), (0)::bigint) AS total_revenue,
    COALESCE(sum(subtotal), (0)::bigint) AS total_subtotal,
    COALESCE(sum(discount), (0)::bigint) AS total_discount,
    COALESCE(sum(tax), (0)::bigint) AS total_tax
   FROM public.sales
  WHERE ((status)::text = 'completed'::text)
  GROUP BY (date((created_at AT TIME ZONE 'Asia/Jakarta'::text))), store_id
  WITH DATA;

CREATE MATERIALIZED VIEW IF NOT EXISTS public.mv_dashboard_totals AS
 SELECT store_id,
    count(*) AS transaction_count,
    COALESCE(sum(total_amount), (0)::bigint) AS total_revenue
   FROM public.sales
  WHERE ((status)::text = 'completed'::text)
  GROUP BY store_id
  WITH DATA;

CREATE MATERIALIZED VIEW IF NOT EXISTS public.mv_hourly_sales AS
 SELECT date_trunc('hour'::text, (created_at AT TIME ZONE 'Asia/Jakarta'::text)) AS sale_hour,
    store_id,
    count(*) AS transaction_count,
    COALESCE(sum(total_amount), (0)::bigint) AS total_revenue
   FROM public.sales
  WHERE ((status)::text = 'completed'::text)
  GROUP BY (date_trunc('hour'::text, (created_at AT TIME ZONE 'Asia/Jakarta'::text))), store_id
  WITH DATA;

CREATE TABLE IF NOT EXISTS public.payment_methods (
    id integer NOT NULL,
    code character varying(30) NOT NULL,
    name character varying(100) NOT NULL,
    is_active boolean DEFAULT true,
    requires_reference boolean DEFAULT false,
    sort_order integer DEFAULT 0 NOT NULL,
    created_at timestamp with time zone DEFAULT now(),
    updated_at timestamp with time zone DEFAULT now()
);

CREATE SEQUENCE IF NOT EXISTS public.payment_methods_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

ALTER SEQUENCE public.payment_methods_id_seq OWNED BY public.payment_methods.id;

CREATE TABLE IF NOT EXISTS public.permissions (
    id integer NOT NULL,
    code character varying(50) NOT NULL,
    name character varying(100) NOT NULL,
    description text,
    created_at timestamp with time zone DEFAULT now()
);

CREATE SEQUENCE IF NOT EXISTS public.permissions_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

ALTER SEQUENCE public.permissions_id_seq OWNED BY public.permissions.id;

CREATE SEQUENCE IF NOT EXISTS public.po_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

CREATE TABLE IF NOT EXISTS public.pricing_rules (
    id integer NOT NULL,
    product_id integer,
    pricing_type character varying(50) NOT NULL,
    name character varying(200),
    minimum_quantity integer DEFAULT 1 NOT NULL,
    priority integer DEFAULT 0 NOT NULL,
    is_active boolean DEFAULT true NOT NULL,
    effective_from timestamp with time zone,
    effective_until timestamp with time zone,
    pricing_method character varying(20) DEFAULT 'fixed_price'::character varying NOT NULL,
    pricing_value numeric(12,2) DEFAULT 0 NOT NULL,
    category_id integer,
    brand_id integer,
    maximum_quantity integer,
    customer_group_id integer,
    store_id integer,
    recurrence_days text[],
    time_from time without time zone,
    time_to time without time zone,
    allow_combine boolean DEFAULT false NOT NULL,
    status character varying(20) DEFAULT 'approved'::character varying NOT NULL,
    created_at timestamp with time zone DEFAULT now(),
    updated_at timestamp with time zone DEFAULT now(),
    CONSTRAINT chk_pricing_status CHECK (((status)::text = ANY (ARRAY[('draft'::character varying)::text, ('pending'::character varying)::text, ('approved'::character varying)::text, ('rejected'::character varying)::text]))),
    CONSTRAINT chk_pricing_target CHECK (((product_id IS NOT NULL) OR (category_id IS NOT NULL) OR (brand_id IS NOT NULL))),
    CONSTRAINT chk_pricing_type CHECK (((pricing_type)::text = ANY (ARRAY[('special_price'::character varying)::text, ('promotion'::character varying)::text]))),
    CONSTRAINT pricing_rules_minimum_quantity_check CHECK ((minimum_quantity >= 1))
);

CREATE SEQUENCE IF NOT EXISTS public.pricing_rules_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

ALTER SEQUENCE public.pricing_rules_id_seq OWNED BY public.pricing_rules.id;

CREATE TABLE IF NOT EXISTS public.product_stock (
    id integer NOT NULL,
    product_id integer NOT NULL,
    warehouse_id integer,
    store_id integer,
    quantity integer DEFAULT 0 NOT NULL,
    reorder_point integer DEFAULT 0 NOT NULL,
    reorder_quantity integer DEFAULT 0 NOT NULL,
    last_restocked_at timestamp with time zone,
    created_at timestamp with time zone DEFAULT now(),
    updated_at timestamp with time zone DEFAULT now(),
    location_id integer,
    CONSTRAINT product_stock_quantity_check CHECK ((quantity >= 0)),
    CONSTRAINT product_stock_reorder_point_check CHECK ((reorder_point >= 0)),
    CONSTRAINT product_stock_reorder_quantity_check CHECK ((reorder_quantity >= 0))
);

CREATE SEQUENCE IF NOT EXISTS public.product_stock_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

ALTER SEQUENCE public.product_stock_id_seq OWNED BY public.product_stock.id;

CREATE TABLE IF NOT EXISTS public.product_suppliers (
    id integer NOT NULL,
    product_id integer NOT NULL,
    supplier_id integer NOT NULL,
    supplier_sku character varying(50),
    unit_cost integer DEFAULT 0,
    lead_time_days integer DEFAULT 0,
    is_preferred boolean DEFAULT false,
    created_at timestamp with time zone DEFAULT now(),
    updated_at timestamp with time zone DEFAULT now(),
    CONSTRAINT product_suppliers_unit_cost_check CHECK ((unit_cost >= 0))
);

CREATE SEQUENCE IF NOT EXISTS public.product_suppliers_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

ALTER SEQUENCE public.product_suppliers_id_seq OWNED BY public.product_suppliers.id;

CREATE TABLE IF NOT EXISTS public.products (
    id integer NOT NULL,
    sku character varying(50) NOT NULL,
    name character varying(200) NOT NULL,
    barcode character varying(50),
    category_id integer,
    brand_id integer,
    description text,
    price integer NOT NULL,
    cost integer DEFAULT 0,
    tax_class_id integer,
    weight_grams integer,
    unit_of_measure_id integer,
    default_discount_percent numeric(5,2) DEFAULT 0,
    status character varying(20) DEFAULT 'active'::character varying NOT NULL,
    store_id integer,
    search_vector tsvector,
    created_at timestamp with time zone DEFAULT now(),
    updated_at timestamp with time zone DEFAULT now(),
    deleted_at timestamp with time zone,
    ownership_type character varying(20) DEFAULT 'store'::character varying,
    CONSTRAINT chk_product_status CHECK (((status)::text = ANY (ARRAY[('draft'::character varying)::text, ('active'::character varying)::text, ('inactive'::character varying)::text, ('discontinued'::character varying)::text, ('archived'::character varying)::text]))),
    CONSTRAINT chk_products_ownership_type CHECK (ownership_type IN ('store', 'consignment')),
    CONSTRAINT products_cost_check CHECK ((cost >= 0)),
    CONSTRAINT products_price_check CHECK ((price >= 0))
);

CREATE SEQUENCE IF NOT EXISTS public.products_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

ALTER SEQUENCE public.products_id_seq OWNED BY public.products.id;

CREATE TABLE IF NOT EXISTS public.purchase_order_items (
    id integer NOT NULL,
    purchase_order_id integer NOT NULL,
    product_id integer NOT NULL,
    qty_ordered integer NOT NULL,
    qty_received integer DEFAULT 0 NOT NULL,
    unit_cost integer NOT NULL,
    discount_amount integer DEFAULT 0 NOT NULL,
    subtotal integer DEFAULT 0 NOT NULL,
    notes text,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    product_name character varying(255) DEFAULT ''::character varying NOT NULL,
    sku character varying(100) DEFAULT ''::character varying,
    barcode character varying(100) DEFAULT ''::character varying,
    uom_id integer,
    uom_name character varying(50) DEFAULT 'pcs'::character varying,
    CONSTRAINT purchase_order_items_qty_ordered_check CHECK ((qty_ordered > 0)),
    CONSTRAINT purchase_order_items_qty_received_check CHECK ((qty_received >= 0)),
    CONSTRAINT purchase_order_items_unit_cost_check CHECK ((unit_cost >= 0))
);

CREATE SEQUENCE IF NOT EXISTS public.purchase_order_items_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

ALTER SEQUENCE public.purchase_order_items_id_seq OWNED BY public.purchase_order_items.id;

CREATE TABLE IF NOT EXISTS public.purchase_orders (
    id integer NOT NULL,
    po_number character varying(30) NOT NULL,
    supplier_id integer NOT NULL,
    store_id integer NOT NULL,
    status character varying(20) DEFAULT 'draft'::character varying NOT NULL,
    expected_date date,
    payment_term character varying(50),
    delivery_address text,
    supplier_reference_number character varying(100),
    notes text,
    confirmed_at timestamp with time zone,
    confirmed_by integer,
    cancelled_at timestamp with time zone,
    cancelled_by integer,
    created_by integer NOT NULL,
    updated_by integer NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    subtotal integer DEFAULT 0 NOT NULL,
    discount_amount integer DEFAULT 0 NOT NULL,
    tax_amount integer DEFAULT 0 NOT NULL,
    grand_total integer DEFAULT 0 NOT NULL,
    approval_status character varying(20) DEFAULT 'pending'::character varying,
    payment_status character varying(20) DEFAULT 'pending'::character varying,
    invoice_status character varying(20) DEFAULT 'pending'::character varying,
    warehouse_id integer,
    currency_code character varying(3) DEFAULT 'IDR'::character varying,
    exchange_rate integer DEFAULT 1,
    approved_by integer,
    approved_at timestamp with time zone
);

CREATE SEQUENCE IF NOT EXISTS public.purchase_orders_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

ALTER SEQUENCE public.purchase_orders_id_seq OWNED BY public.purchase_orders.id;

CREATE TABLE IF NOT EXISTS public.refresh_tokens (
    id integer NOT NULL,
    user_id integer NOT NULL,
    token_hash text NOT NULL,
    expires_at timestamp with time zone NOT NULL,
    created_at timestamp with time zone DEFAULT now()
);

CREATE SEQUENCE IF NOT EXISTS public.refresh_tokens_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

ALTER SEQUENCE public.refresh_tokens_id_seq OWNED BY public.refresh_tokens.id;

CREATE TABLE IF NOT EXISTS public.role_permissions (
    role_id integer NOT NULL,
    permission_id integer NOT NULL
);

CREATE TABLE IF NOT EXISTS public.roles (
    id integer NOT NULL,
    name character varying(50) NOT NULL,
    description text,
    is_system boolean DEFAULT false,
    created_at timestamp with time zone DEFAULT now()
);

CREATE SEQUENCE IF NOT EXISTS public.roles_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

ALTER SEQUENCE public.roles_id_seq OWNED BY public.roles.id;

CREATE TABLE IF NOT EXISTS public.sale_items (
    id integer NOT NULL,
    sale_id integer NOT NULL,
    product_id integer NOT NULL,
    quantity integer NOT NULL,
    unit_price integer NOT NULL,
    subtotal integer NOT NULL,
    dpp_amount integer DEFAULT 0 NOT NULL,
    tax_amount integer DEFAULT 0 NOT NULL,
    pricing_rule_id integer,
    pricing_rule_name character varying(200),
    pricing_rule_type character varying(50),
    pricing_type character varying(50),
    original_price integer DEFAULT 0 NOT NULL,
    cost integer DEFAULT 0 NOT NULL,
    tax_class_id integer,
    tax_rate numeric(5,2),
    snapshot_created_at timestamp with time zone DEFAULT now() NOT NULL,
    product_name character varying(200),
    CONSTRAINT sale_items_quantity_check CHECK ((quantity > 0)),
    CONSTRAINT sale_items_subtotal_check CHECK ((subtotal >= 0)),
    CONSTRAINT sale_items_unit_price_check CHECK ((unit_price >= 0))
);

CREATE SEQUENCE IF NOT EXISTS public.sale_items_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

ALTER SEQUENCE public.sale_items_id_seq OWNED BY public.sale_items.id;

CREATE TABLE IF NOT EXISTS public.sale_payments (
    id integer NOT NULL,
    sale_id integer NOT NULL,
    payment_method_id integer NOT NULL,
    payment_method_code character varying(30) NOT NULL,
    amount integer NOT NULL,
    reference_number character varying(100),
    created_at timestamp with time zone DEFAULT now(),
    CONSTRAINT sale_payments_amount_check CHECK ((amount > 0))
);

CREATE SEQUENCE IF NOT EXISTS public.sale_payments_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

ALTER SEQUENCE public.sale_payments_id_seq OWNED BY public.sale_payments.id;

CREATE SEQUENCE IF NOT EXISTS public.sales_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

ALTER SEQUENCE public.sales_id_seq OWNED BY public.sales.id;

CREATE TABLE IF NOT EXISTS public.schema_migrations (
    filename character varying(255) NOT NULL,
    applied_at timestamp with time zone DEFAULT now() NOT NULL
);

CREATE TABLE IF NOT EXISTS public.shifts (
    id integer NOT NULL,
    user_id integer NOT NULL,
    store_id integer,
    status character varying(20) DEFAULT 'open'::character varying NOT NULL,
    opening_balance integer DEFAULT 0 NOT NULL,
    closing_balance integer,
    cash_sales integer DEFAULT 0 NOT NULL,
    non_cash_sales integer DEFAULT 0 NOT NULL,
    total_sales integer DEFAULT 0 NOT NULL,
    transaction_count integer DEFAULT 0 NOT NULL,
    discrepancy integer,
    notes text,
    opened_at timestamp with time zone DEFAULT now() NOT NULL,
    closed_at timestamp with time zone,
    needs_review boolean DEFAULT false NOT NULL,
    reviewed_by integer,
    reviewed_at timestamp with time zone,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT chk_shift_status CHECK (((status)::text = ANY (ARRAY[('open'::character varying)::text, ('closed'::character varying)::text])))
);

CREATE SEQUENCE IF NOT EXISTS public.shifts_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

ALTER SEQUENCE public.shifts_id_seq OWNED BY public.shifts.id;

CREATE SEQUENCE IF NOT EXISTS public.sku_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

CREATE SEQUENCE IF NOT EXISTS public.so_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

CREATE TABLE IF NOT EXISTS public.stock_opname_assignments (
    id integer NOT NULL,
    stock_opname_id integer NOT NULL,
    user_id integer NOT NULL,
    role character varying(20) NOT NULL,
    assigned_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT chk_stock_opname_assignment_role CHECK (((role)::text = ANY (ARRAY[('counter'::character varying)::text, ('supervisor'::character varying)::text])))
);

CREATE SEQUENCE IF NOT EXISTS public.stock_opname_assignments_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

ALTER SEQUENCE public.stock_opname_assignments_id_seq OWNED BY public.stock_opname_assignments.id;

CREATE TABLE IF NOT EXISTS public.stock_opname_counts (
    id integer NOT NULL,
    stock_opname_item_id integer NOT NULL,
    count_sequence integer NOT NULL,
    physical_qty numeric(18,4) NOT NULL,
    counted_by integer NOT NULL,
    counted_at timestamp with time zone DEFAULT now() NOT NULL,
    remarks text,
    CONSTRAINT stock_opname_counts_count_sequence_check CHECK ((count_sequence >= 1)),
    CONSTRAINT stock_opname_counts_physical_qty_check CHECK ((physical_qty >= (0)::numeric))
);

CREATE SEQUENCE IF NOT EXISTS public.stock_opname_counts_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

ALTER SEQUENCE public.stock_opname_counts_id_seq OWNED BY public.stock_opname_counts.id;

CREATE TABLE IF NOT EXISTS public.stock_opname_items (
    id integer NOT NULL,
    stock_opname_id integer NOT NULL,
    product_id integer NOT NULL,
    opening_qty numeric(18,4) DEFAULT 0 NOT NULL,
    expected_qty numeric(18,4) DEFAULT 0 NOT NULL,
    physical_qty numeric(18,4) DEFAULT 0 NOT NULL,
    difference_qty numeric(18,4) DEFAULT 0 NOT NULL,
    adjustment_qty numeric(18,4) DEFAULT 0 NOT NULL,
    status character varying(20) DEFAULT 'pending'::character varying NOT NULL,
    product_name character varying(255) DEFAULT ''::character varying NOT NULL,
    sku character varying(100) DEFAULT ''::character varying,
    barcode character varying(100) DEFAULT ''::character varying,
    uom_name character varying(50) DEFAULT 'pcs'::character varying,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    reason text,
    warehouse_id integer,
    store_id integer,
    CONSTRAINT chk_stock_opname_item_status CHECK (((status)::text = ANY (ARRAY[('pending'::character varying)::text, ('counted'::character varying)::text]))),
    CONSTRAINT stock_opname_items_physical_qty_check CHECK ((physical_qty >= (0)::numeric))
);

CREATE SEQUENCE IF NOT EXISTS public.stock_opname_items_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

ALTER SEQUENCE public.stock_opname_items_id_seq OWNED BY public.stock_opname_items.id;

CREATE TABLE IF NOT EXISTS public.stock_opname_recount_requests (
    id bigint NOT NULL,
    stock_opname_id integer NOT NULL,
    requested_by integer NOT NULL,
    reason text,
    created_at timestamp with time zone DEFAULT now() NOT NULL
);

CREATE SEQUENCE IF NOT EXISTS public.stock_opname_recount_requests_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

ALTER SEQUENCE public.stock_opname_recount_requests_id_seq OWNED BY public.stock_opname_recount_requests.id;

CREATE TABLE IF NOT EXISTS public.stock_opname_session_scopes (
    id bigint NOT NULL,
    stock_opname_id integer NOT NULL,
    scope_type character varying(30) NOT NULL,
    scope_id bigint,
    scope_name character varying(255) DEFAULT ''::character varying,
    scope_data jsonb,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT chk_so_scope_type CHECK (((scope_type)::text = ANY (ARRAY[('store'::character varying)::text, ('warehouse'::character varying)::text, ('category'::character varying)::text, ('brand'::character varying)::text, ('supplier'::character varying)::text, ('product'::character varying)::text, ('manual'::character varying)::text, ('location'::character varying)::text])))
);

CREATE SEQUENCE IF NOT EXISTS public.stock_opname_session_scopes_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

ALTER SEQUENCE public.stock_opname_session_scopes_id_seq OWNED BY public.stock_opname_session_scopes.id;

CREATE TABLE IF NOT EXISTS public.stock_opnames (
    id integer NOT NULL,
    session_number character varying(30) NOT NULL,
    scope_type character varying(20) NOT NULL,
    scope_id bigint NOT NULL,
    warehouse_id bigint,
    blind_count boolean DEFAULT false NOT NULL,
    status character varying(20) DEFAULT 'draft'::character varying NOT NULL,
    created_by integer NOT NULL,
    approved_by integer,
    approved_at timestamp with time zone,
    cancelled_at timestamp with time zone,
    deleted_at timestamp with time zone,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    store_id integer,
    opened_by integer,
    opened_at timestamp with time zone,
    verified_by integer,
    verified_at timestamp with time zone,
    posted_by integer,
    posted_at timestamp with time zone,
    closed_by integer,
    closed_at timestamp with time zone,
    scope_name character varying(255) DEFAULT ''::character varying,
    title character varying(255) DEFAULT ''::character varying,
    notes text,
    total_difference numeric(18,4) DEFAULT 0 NOT NULL,
    total_adjustment numeric(18,4) DEFAULT 0 NOT NULL,
    location_id integer,
    CONSTRAINT chk_stock_opname_scope_type CHECK (((scope_type)::text = ANY (ARRAY[('store'::character varying)::text, ('warehouse'::character varying)::text, ('category'::character varying)::text, ('brand'::character varying)::text, ('supplier'::character varying)::text, ('product'::character varying)::text, ('manual'::character varying)::text, ('location'::character varying)::text]))),
    CONSTRAINT chk_stock_opname_status CHECK (((status)::text = ANY (ARRAY[('draft'::character varying)::text, ('open'::character varying)::text, ('counting'::character varying)::text, ('verification'::character varying)::text, ('needs_recount'::character varying)::text, ('approved'::character varying)::text, ('posted'::character varying)::text, ('closed'::character varying)::text, ('cancelled'::character varying)::text])))
);

CREATE SEQUENCE IF NOT EXISTS public.stock_opnames_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

ALTER SEQUENCE public.stock_opnames_id_seq OWNED BY public.stock_opnames.id;

CREATE TABLE IF NOT EXISTS public.storage_locations (
    id integer NOT NULL,
    code character varying(50) NOT NULL,
    name character varying(100) NOT NULL,
    warehouse_id integer,
    store_id integer,
    notes text,
    is_active boolean DEFAULT true NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT chk_storage_locations_scope CHECK (((warehouse_id IS NOT NULL) OR (store_id IS NOT NULL)))
);

CREATE SEQUENCE IF NOT EXISTS public.storage_locations_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

ALTER SEQUENCE public.storage_locations_id_seq OWNED BY public.storage_locations.id;

CREATE TABLE IF NOT EXISTS public.stores (
    id integer NOT NULL,
    name character varying(100) NOT NULL,
    address text,
    phone character varying(20),
    is_active boolean DEFAULT true,
    created_at timestamp with time zone DEFAULT now()
);

CREATE SEQUENCE IF NOT EXISTS public.stores_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

ALTER SEQUENCE public.stores_id_seq OWNED BY public.stores.id;

CREATE SEQUENCE IF NOT EXISTS public.supplier_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

CREATE TABLE IF NOT EXISTS public.suppliers (
    id integer NOT NULL,
    name character varying(200) NOT NULL,
    code character varying(50) NOT NULL,
    contact_name character varying(200),
    phone character varying(50),
    email character varying(200),
    address text,
    notes text,
    is_active boolean DEFAULT true,
    store_id integer,
    created_at timestamp with time zone DEFAULT now(),
    updated_at timestamp with time zone DEFAULT now(),
    deleted_at timestamp with time zone,
    is_consignment boolean DEFAULT false
);

CREATE SEQUENCE IF NOT EXISTS public.suppliers_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

ALTER SEQUENCE public.suppliers_id_seq OWNED BY public.suppliers.id;

CREATE TABLE IF NOT EXISTS public.tax_classes (
    id integer NOT NULL,
    name character varying(100) NOT NULL,
    rate_percent numeric(5,2) NOT NULL,
    description text,
    is_active boolean DEFAULT true,
    created_at timestamp with time zone DEFAULT now()
);

CREATE SEQUENCE IF NOT EXISTS public.tax_classes_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

ALTER SEQUENCE public.tax_classes_id_seq OWNED BY public.tax_classes.id;

CREATE TABLE IF NOT EXISTS public.units_of_measure (
    id integer NOT NULL,
    code character varying(10) NOT NULL,
    name character varying(100) NOT NULL,
    description text,
    is_active boolean DEFAULT true,
    created_at timestamp with time zone DEFAULT now()
);

CREATE SEQUENCE IF NOT EXISTS public.units_of_measure_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

ALTER SEQUENCE public.units_of_measure_id_seq OWNED BY public.units_of_measure.id;

CREATE TABLE IF NOT EXISTS public.users (
    id integer NOT NULL,
    username character varying(50) NOT NULL,
    email character varying(100) NOT NULL,
    password_hash text NOT NULL,
    role_id integer NOT NULL,
    store_id integer,
    is_active boolean DEFAULT true,
    last_login timestamp with time zone,
    created_at timestamp with time zone DEFAULT now(),
    updated_at timestamp with time zone DEFAULT now(),
    deleted_at timestamp with time zone,
    reports_to integer,
    language character varying(5) DEFAULT 'id'::character varying,
    theme character varying(10) DEFAULT 'light'::character varying,
    must_change_password boolean DEFAULT false NOT NULL
);

CREATE SEQUENCE IF NOT EXISTS public.users_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

ALTER SEQUENCE public.users_id_seq OWNED BY public.users.id;

CREATE OR REPLACE VIEW public.v_products_full AS
 SELECT p.id,
    p.sku,
    p.name,
    p.barcode,
    p.category_id,
    c.name AS category_name,
    p.price,
    COALESCE(p.cost, 0) AS cost,
    COALESCE(ps.quantity, 0) AS stock,
    p.status,
    p.store_id,
    p.brand_id,
    b.name AS brand_name,
    p.unit_of_measure_id,
    u.name AS unit_of_measure,
    p.weight_grams,
    p.description,
    p.tax_class_id,
    tc.rate_percent AS tax_rate,
    p.search_vector,
    p.created_at,
    p.updated_at,
    ps_preferred.supplier_id,
    ps_preferred.supplier_name,
    p.ownership_type
   FROM ((((((public.products p
     LEFT JOIN public.categories c ON ((p.category_id = c.id)))
     LEFT JOIN public.brands b ON ((p.brand_id = b.id)))
     LEFT JOIN public.units_of_measure u ON ((p.unit_of_measure_id = u.id)))
     LEFT JOIN LATERAL ( SELECT product_stock.quantity
           FROM public.product_stock
          WHERE (product_stock.product_id = p.id)
          ORDER BY ((product_stock.warehouse_id IS NULL) AND (product_stock.store_id IS NULL)) DESC
         LIMIT 1) ps ON (true))
     LEFT JOIN public.tax_classes tc ON ((tc.id = p.tax_class_id)))
     LEFT JOIN LATERAL ( SELECT s.id AS supplier_id,
            s.name AS supplier_name
           FROM (public.product_suppliers ps_1
             JOIN public.suppliers s ON (((ps_1.supplier_id = s.id) AND (s.deleted_at IS NULL))))
          WHERE ((ps_1.product_id = p.id) AND (ps_1.is_preferred = true))
         LIMIT 1) ps_preferred ON (true))
  WHERE (p.deleted_at IS NULL);

CREATE TABLE IF NOT EXISTS public.warehouses (
    id integer NOT NULL,
    name character varying(100) NOT NULL,
    code character varying(20) NOT NULL,
    address text,
    store_id integer,
    is_active boolean DEFAULT true,
    created_at timestamp with time zone DEFAULT now()
);

CREATE SEQUENCE IF NOT EXISTS public.warehouses_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

ALTER SEQUENCE public.warehouses_id_seq OWNED BY public.warehouses.id;

-- ============================================================================
-- Column defaults
-- ============================================================================

ALTER TABLE ONLY public.audit_logs ALTER COLUMN id SET DEFAULT nextval('public.audit_logs_id_seq'::regclass);

ALTER TABLE ONLY public.brands ALTER COLUMN id SET DEFAULT nextval('public.brands_id_seq'::regclass);

ALTER TABLE ONLY public.cart_items ALTER COLUMN id SET DEFAULT nextval('public.cart_items_id_seq'::regclass);

ALTER TABLE ONLY public.cart_sessions ALTER COLUMN id SET DEFAULT nextval('public.cart_sessions_id_seq'::regclass);

ALTER TABLE ONLY public.cash_movements ALTER COLUMN id SET DEFAULT nextval('public.cash_movements_id_seq'::regclass);

ALTER TABLE ONLY public.categories ALTER COLUMN id SET DEFAULT nextval('public.categories_id_seq'::regclass);

ALTER TABLE ONLY public.consignment_arrangements ALTER COLUMN id SET DEFAULT nextval('public.consignment_arrangements_id_seq'::regclass);

ALTER TABLE ONLY public.consignment_payouts ALTER COLUMN id SET DEFAULT nextval('public.consignment_payouts_id_seq'::regclass);

ALTER TABLE ONLY public.consignment_pending_returns ALTER COLUMN id SET DEFAULT nextval('public.consignment_pending_returns_id_seq'::regclass);

ALTER TABLE ONLY public.consignment_receipt_edits ALTER COLUMN id SET DEFAULT nextval('public.consignment_receipt_edits_id_seq'::regclass);

ALTER TABLE ONLY public.consignment_receipt_items ALTER COLUMN id SET DEFAULT nextval('public.consignment_receipt_items_id_seq'::regclass);

ALTER TABLE ONLY public.consignment_receipts ALTER COLUMN id SET DEFAULT nextval('public.consignment_receipts_id_seq'::regclass);

ALTER TABLE ONLY public.consignment_return_items ALTER COLUMN id SET DEFAULT nextval('public.consignment_return_items_id_seq'::regclass);

ALTER TABLE ONLY public.consignment_returns ALTER COLUMN id SET DEFAULT nextval('public.consignment_returns_id_seq'::regclass);

ALTER TABLE ONLY public.consignment_sale_items ALTER COLUMN id SET DEFAULT nextval('public.consignment_sale_items_id_seq'::regclass);

ALTER TABLE ONLY public.consignment_settlement_items ALTER COLUMN id SET DEFAULT nextval('public.consignment_settlement_items_id_seq'::regclass);

ALTER TABLE ONLY public.consignment_settlements ALTER COLUMN id SET DEFAULT nextval('public.consignment_settlements_id_seq'::regclass);

ALTER TABLE ONLY public.consignment_stock ALTER COLUMN id SET DEFAULT nextval('public.consignment_stock_id_seq'::regclass);

ALTER TABLE ONLY public.consignment_terms ALTER COLUMN id SET DEFAULT nextval('public.consignment_terms_id_seq'::regclass);

ALTER TABLE ONLY public.customer_groups ALTER COLUMN id SET DEFAULT nextval('public.customer_groups_id_seq'::regclass);

ALTER TABLE ONLY public.customers ALTER COLUMN id SET DEFAULT nextval('public.customers_id_seq'::regclass);

ALTER TABLE ONLY public.dead_letter_events ALTER COLUMN id SET DEFAULT nextval('public.dead_letter_events_id_seq'::regclass);

ALTER TABLE ONLY public.goods_receipt_items ALTER COLUMN id SET DEFAULT nextval('public.goods_receipt_items_id_seq'::regclass);

ALTER TABLE ONLY public.goods_receipts ALTER COLUMN id SET DEFAULT nextval('public.goods_receipts_id_seq'::regclass);

ALTER TABLE ONLY public.import_errors ALTER COLUMN id SET DEFAULT nextval('public.import_errors_id_seq'::regclass);

ALTER TABLE ONLY public.import_jobs ALTER COLUMN id SET DEFAULT nextval('public.import_jobs_id_seq'::regclass);

ALTER TABLE ONLY public.import_rows ALTER COLUMN id SET DEFAULT nextval('public.import_rows_id_seq'::regclass);

ALTER TABLE ONLY public.import_snapshots ALTER COLUMN id SET DEFAULT nextval('public.import_snapshots_id_seq'::regclass);

ALTER TABLE ONLY public.inventory_adjustment_items ALTER COLUMN id SET DEFAULT nextval('public.inventory_adjustment_items_id_seq'::regclass);

ALTER TABLE ONLY public.inventory_adjustments ALTER COLUMN id SET DEFAULT nextval('public.inventory_adjustments_id_seq'::regclass);

ALTER TABLE ONLY public.inventory_movements ALTER COLUMN id SET DEFAULT nextval('public.inventory_movements_id_seq'::regclass);

ALTER TABLE ONLY public.payment_methods ALTER COLUMN id SET DEFAULT nextval('public.payment_methods_id_seq'::regclass);

ALTER TABLE ONLY public.permissions ALTER COLUMN id SET DEFAULT nextval('public.permissions_id_seq'::regclass);

ALTER TABLE ONLY public.pricing_rules ALTER COLUMN id SET DEFAULT nextval('public.pricing_rules_id_seq'::regclass);

ALTER TABLE ONLY public.product_stock ALTER COLUMN id SET DEFAULT nextval('public.product_stock_id_seq'::regclass);

ALTER TABLE ONLY public.product_suppliers ALTER COLUMN id SET DEFAULT nextval('public.product_suppliers_id_seq'::regclass);

ALTER TABLE ONLY public.products ALTER COLUMN id SET DEFAULT nextval('public.products_id_seq'::regclass);

ALTER TABLE ONLY public.purchase_order_items ALTER COLUMN id SET DEFAULT nextval('public.purchase_order_items_id_seq'::regclass);

ALTER TABLE ONLY public.purchase_orders ALTER COLUMN id SET DEFAULT nextval('public.purchase_orders_id_seq'::regclass);

ALTER TABLE ONLY public.refresh_tokens ALTER COLUMN id SET DEFAULT nextval('public.refresh_tokens_id_seq'::regclass);

ALTER TABLE ONLY public.roles ALTER COLUMN id SET DEFAULT nextval('public.roles_id_seq'::regclass);

ALTER TABLE ONLY public.sale_items ALTER COLUMN id SET DEFAULT nextval('public.sale_items_id_seq'::regclass);

ALTER TABLE ONLY public.sale_payments ALTER COLUMN id SET DEFAULT nextval('public.sale_payments_id_seq'::regclass);

ALTER TABLE ONLY public.sales ALTER COLUMN id SET DEFAULT nextval('public.sales_id_seq'::regclass);

ALTER TABLE ONLY public.shifts ALTER COLUMN id SET DEFAULT nextval('public.shifts_id_seq'::regclass);

ALTER TABLE ONLY public.stock_opname_assignments ALTER COLUMN id SET DEFAULT nextval('public.stock_opname_assignments_id_seq'::regclass);

ALTER TABLE ONLY public.stock_opname_counts ALTER COLUMN id SET DEFAULT nextval('public.stock_opname_counts_id_seq'::regclass);

ALTER TABLE ONLY public.stock_opname_items ALTER COLUMN id SET DEFAULT nextval('public.stock_opname_items_id_seq'::regclass);

ALTER TABLE ONLY public.stock_opname_recount_requests ALTER COLUMN id SET DEFAULT nextval('public.stock_opname_recount_requests_id_seq'::regclass);

ALTER TABLE ONLY public.stock_opname_session_scopes ALTER COLUMN id SET DEFAULT nextval('public.stock_opname_session_scopes_id_seq'::regclass);

ALTER TABLE ONLY public.stock_opnames ALTER COLUMN id SET DEFAULT nextval('public.stock_opnames_id_seq'::regclass);

ALTER TABLE ONLY public.storage_locations ALTER COLUMN id SET DEFAULT nextval('public.storage_locations_id_seq'::regclass);

ALTER TABLE ONLY public.stores ALTER COLUMN id SET DEFAULT nextval('public.stores_id_seq'::regclass);

ALTER TABLE ONLY public.suppliers ALTER COLUMN id SET DEFAULT nextval('public.suppliers_id_seq'::regclass);

ALTER TABLE ONLY public.tax_classes ALTER COLUMN id SET DEFAULT nextval('public.tax_classes_id_seq'::regclass);

ALTER TABLE ONLY public.units_of_measure ALTER COLUMN id SET DEFAULT nextval('public.units_of_measure_id_seq'::regclass);

ALTER TABLE ONLY public.users ALTER COLUMN id SET DEFAULT nextval('public.users_id_seq'::regclass);

ALTER TABLE ONLY public.warehouses ALTER COLUMN id SET DEFAULT nextval('public.warehouses_id_seq'::regclass);

-- ============================================================================
-- Constraints
-- ============================================================================

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conrelid = 'public.app_settings'::regclass AND conname = 'app_settings_pkey'
    ) THEN
        ALTER TABLE ONLY public.app_settings ADD CONSTRAINT app_settings_pkey PRIMARY KEY (key);
    END IF;
END;
$$;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conrelid = 'public.audit_logs'::regclass AND conname = 'audit_logs_pkey'
    ) THEN
        ALTER TABLE ONLY public.audit_logs ADD CONSTRAINT audit_logs_pkey PRIMARY KEY (id);
    END IF;
END;
$$;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conrelid = 'public.brands'::regclass AND conname = 'brands_name_key'
    ) THEN
        ALTER TABLE ONLY public.brands ADD CONSTRAINT brands_name_key UNIQUE (name);
    END IF;
END;
$$;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conrelid = 'public.brands'::regclass AND conname = 'brands_pkey'
    ) THEN
        ALTER TABLE ONLY public.brands ADD CONSTRAINT brands_pkey PRIMARY KEY (id);
    END IF;
END;
$$;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conrelid = 'public.cart_items'::regclass AND conname = 'cart_items_pkey'
    ) THEN
        ALTER TABLE ONLY public.cart_items ADD CONSTRAINT cart_items_pkey PRIMARY KEY (id);
    END IF;
END;
$$;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conrelid = 'public.cart_sessions'::regclass AND conname = 'cart_sessions_pkey'
    ) THEN
        ALTER TABLE ONLY public.cart_sessions ADD CONSTRAINT cart_sessions_pkey PRIMARY KEY (id);
    END IF;
END;
$$;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conrelid = 'public.cash_movements'::regclass AND conname = 'cash_movements_pkey'
    ) THEN
        ALTER TABLE ONLY public.cash_movements ADD CONSTRAINT cash_movements_pkey PRIMARY KEY (id);
    END IF;
END;
$$;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conrelid = 'public.categories'::regclass AND conname = 'categories_name_key'
    ) THEN
        ALTER TABLE ONLY public.categories ADD CONSTRAINT categories_name_key UNIQUE (name);
    END IF;
END;
$$;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conrelid = 'public.categories'::regclass AND conname = 'categories_pkey'
    ) THEN
        ALTER TABLE ONLY public.categories ADD CONSTRAINT categories_pkey PRIMARY KEY (id);
    END IF;
END;
$$;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conrelid = 'public.categories'::regclass AND conname = 'categories_slug_key'
    ) THEN
        ALTER TABLE ONLY public.categories ADD CONSTRAINT categories_slug_key UNIQUE (slug);
    END IF;
END;
$$;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conrelid = 'public.consignment_arrangements'::regclass AND conname = 'consignment_arrangements_pkey'
    ) THEN
        ALTER TABLE ONLY public.consignment_arrangements ADD CONSTRAINT consignment_arrangements_pkey PRIMARY KEY (id);
    END IF;
END;
$$;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conrelid = 'public.consignment_payouts'::regclass AND conname = 'consignment_payouts_payout_number_key'
    ) THEN
        ALTER TABLE ONLY public.consignment_payouts ADD CONSTRAINT consignment_payouts_payout_number_key UNIQUE (payout_number);
    END IF;
END;
$$;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conrelid = 'public.consignment_payouts'::regclass AND conname = 'consignment_payouts_pkey'
    ) THEN
        ALTER TABLE ONLY public.consignment_payouts ADD CONSTRAINT consignment_payouts_pkey PRIMARY KEY (id);
    END IF;
END;
$$;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conrelid = 'public.consignment_pending_returns'::regclass AND conname = 'consignment_pending_returns_pkey'
    ) THEN
        ALTER TABLE ONLY public.consignment_pending_returns ADD CONSTRAINT consignment_pending_returns_pkey PRIMARY KEY (id);
    END IF;
END;
$$;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conrelid = 'public.consignment_receipt_edits'::regclass AND conname = 'consignment_receipt_edits_pkey'
    ) THEN
        ALTER TABLE ONLY public.consignment_receipt_edits ADD CONSTRAINT consignment_receipt_edits_pkey PRIMARY KEY (id);
    END IF;
END;
$$;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conrelid = 'public.consignment_receipt_items'::regclass AND conname = 'consignment_receipt_items_pkey'
    ) THEN
        ALTER TABLE ONLY public.consignment_receipt_items ADD CONSTRAINT consignment_receipt_items_pkey PRIMARY KEY (id);
    END IF;
END;
$$;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conrelid = 'public.consignment_receipts'::regclass AND conname = 'consignment_receipts_pkey'
    ) THEN
        ALTER TABLE ONLY public.consignment_receipts ADD CONSTRAINT consignment_receipts_pkey PRIMARY KEY (id);
    END IF;
END;
$$;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conrelid = 'public.consignment_receipts'::regclass AND conname = 'consignment_receipts_receipt_number_key'
    ) THEN
        ALTER TABLE ONLY public.consignment_receipts ADD CONSTRAINT consignment_receipts_receipt_number_key UNIQUE (receipt_number);
    END IF;
END;
$$;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conrelid = 'public.consignment_return_items'::regclass AND conname = 'consignment_return_items_pkey'
    ) THEN
        ALTER TABLE ONLY public.consignment_return_items ADD CONSTRAINT consignment_return_items_pkey PRIMARY KEY (id);
    END IF;
END;
$$;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conrelid = 'public.consignment_returns'::regclass AND conname = 'consignment_returns_pkey'
    ) THEN
        ALTER TABLE ONLY public.consignment_returns ADD CONSTRAINT consignment_returns_pkey PRIMARY KEY (id);
    END IF;
END;
$$;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conrelid = 'public.consignment_returns'::regclass AND conname = 'consignment_returns_return_number_key'
    ) THEN
        ALTER TABLE ONLY public.consignment_returns ADD CONSTRAINT consignment_returns_return_number_key UNIQUE (return_number);
    END IF;
END;
$$;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conrelid = 'public.consignment_sale_items'::regclass AND conname = 'consignment_sale_items_pkey'
    ) THEN
        ALTER TABLE ONLY public.consignment_sale_items ADD CONSTRAINT consignment_sale_items_pkey PRIMARY KEY (id);
    END IF;
END;
$$;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conrelid = 'public.consignment_settlement_items'::regclass AND conname = 'consignment_settlement_items_pkey'
    ) THEN
        ALTER TABLE ONLY public.consignment_settlement_items ADD CONSTRAINT consignment_settlement_items_pkey PRIMARY KEY (id);
    END IF;
END;
$$;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conrelid = 'public.consignment_settlements'::regclass AND conname = 'consignment_settlements_pkey'
    ) THEN
        ALTER TABLE ONLY public.consignment_settlements ADD CONSTRAINT consignment_settlements_pkey PRIMARY KEY (id);
    END IF;
END;
$$;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conrelid = 'public.consignment_settlements'::regclass AND conname = 'consignment_settlements_settlement_number_key'
    ) THEN
        ALTER TABLE ONLY public.consignment_settlements ADD CONSTRAINT consignment_settlements_settlement_number_key UNIQUE (settlement_number);
    END IF;
END;
$$;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conrelid = 'public.consignment_stock'::regclass AND conname = 'consignment_stock_pkey'
    ) THEN
        ALTER TABLE ONLY public.consignment_stock ADD CONSTRAINT consignment_stock_pkey PRIMARY KEY (id);
    END IF;
END;
$$;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conrelid = 'public.consignment_terms'::regclass AND conname = 'consignment_terms_pkey'
    ) THEN
        ALTER TABLE ONLY public.consignment_terms ADD CONSTRAINT consignment_terms_pkey PRIMARY KEY (id);
    END IF;
END;
$$;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conrelid = 'public.customer_groups'::regclass AND conname = 'customer_groups_name_key'
    ) THEN
        ALTER TABLE ONLY public.customer_groups ADD CONSTRAINT customer_groups_name_key UNIQUE (name);
    END IF;
END;
$$;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conrelid = 'public.customer_groups'::regclass AND conname = 'customer_groups_pkey'
    ) THEN
        ALTER TABLE ONLY public.customer_groups ADD CONSTRAINT customer_groups_pkey PRIMARY KEY (id);
    END IF;
END;
$$;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conrelid = 'public.customers'::regclass AND conname = 'customers_phone_key'
    ) THEN
        ALTER TABLE ONLY public.customers ADD CONSTRAINT customers_phone_key UNIQUE (phone);
    END IF;
END;
$$;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conrelid = 'public.customers'::regclass AND conname = 'customers_pkey'
    ) THEN
        ALTER TABLE ONLY public.customers ADD CONSTRAINT customers_pkey PRIMARY KEY (id);
    END IF;
END;
$$;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conrelid = 'public.dead_letter_events'::regclass AND conname = 'dead_letter_events_pkey'
    ) THEN
        ALTER TABLE ONLY public.dead_letter_events ADD CONSTRAINT dead_letter_events_pkey PRIMARY KEY (id);
    END IF;
END;
$$;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conrelid = 'public.goods_receipt_items'::regclass AND conname = 'goods_receipt_items_pkey'
    ) THEN
        ALTER TABLE ONLY public.goods_receipt_items ADD CONSTRAINT goods_receipt_items_pkey PRIMARY KEY (id);
    END IF;
END;
$$;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conrelid = 'public.goods_receipts'::regclass AND conname = 'goods_receipts_gr_number_key'
    ) THEN
        ALTER TABLE ONLY public.goods_receipts ADD CONSTRAINT goods_receipts_gr_number_key UNIQUE (gr_number);
    END IF;
END;
$$;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conrelid = 'public.goods_receipts'::regclass AND conname = 'goods_receipts_pkey'
    ) THEN
        ALTER TABLE ONLY public.goods_receipts ADD CONSTRAINT goods_receipts_pkey PRIMARY KEY (id);
    END IF;
END;
$$;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conrelid = 'public.import_errors'::regclass AND conname = 'import_errors_pkey'
    ) THEN
        ALTER TABLE ONLY public.import_errors ADD CONSTRAINT import_errors_pkey PRIMARY KEY (id);
    END IF;
END;
$$;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conrelid = 'public.import_jobs'::regclass AND conname = 'import_jobs_pkey'
    ) THEN
        ALTER TABLE ONLY public.import_jobs ADD CONSTRAINT import_jobs_pkey PRIMARY KEY (id);
    END IF;
END;
$$;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conrelid = 'public.import_rows'::regclass AND conname = 'import_rows_pkey'
    ) THEN
        ALTER TABLE ONLY public.import_rows ADD CONSTRAINT import_rows_pkey PRIMARY KEY (id);
    END IF;
END;
$$;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conrelid = 'public.import_snapshots'::regclass AND conname = 'import_snapshots_pkey'
    ) THEN
        ALTER TABLE ONLY public.import_snapshots ADD CONSTRAINT import_snapshots_pkey PRIMARY KEY (id);
    END IF;
END;
$$;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conrelid = 'public.inventory_adjustment_items'::regclass AND conname = 'inventory_adjustment_items_pkey'
    ) THEN
        ALTER TABLE ONLY public.inventory_adjustment_items ADD CONSTRAINT inventory_adjustment_items_pkey PRIMARY KEY (id);
    END IF;
END;
$$;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conrelid = 'public.inventory_adjustments'::regclass AND conname = 'inventory_adjustments_adjustment_number_key'
    ) THEN
        ALTER TABLE ONLY public.inventory_adjustments ADD CONSTRAINT inventory_adjustments_adjustment_number_key UNIQUE (adjustment_number);
    END IF;
END;
$$;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conrelid = 'public.inventory_adjustments'::regclass AND conname = 'inventory_adjustments_pkey'
    ) THEN
        ALTER TABLE ONLY public.inventory_adjustments ADD CONSTRAINT inventory_adjustments_pkey PRIMARY KEY (id);
    END IF;
END;
$$;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conrelid = 'public.inventory_movements'::regclass AND conname = 'inventory_movements_pkey'
    ) THEN
        ALTER TABLE ONLY public.inventory_movements ADD CONSTRAINT inventory_movements_pkey PRIMARY KEY (id);
    END IF;
END;
$$;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conrelid = 'public.payment_methods'::regclass AND conname = 'payment_methods_code_key'
    ) THEN
        ALTER TABLE ONLY public.payment_methods ADD CONSTRAINT payment_methods_code_key UNIQUE (code);
    END IF;
END;
$$;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conrelid = 'public.payment_methods'::regclass AND conname = 'payment_methods_pkey'
    ) THEN
        ALTER TABLE ONLY public.payment_methods ADD CONSTRAINT payment_methods_pkey PRIMARY KEY (id);
    END IF;
END;
$$;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conrelid = 'public.permissions'::regclass AND conname = 'permissions_code_key'
    ) THEN
        ALTER TABLE ONLY public.permissions ADD CONSTRAINT permissions_code_key UNIQUE (code);
    END IF;
END;
$$;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conrelid = 'public.permissions'::regclass AND conname = 'permissions_pkey'
    ) THEN
        ALTER TABLE ONLY public.permissions ADD CONSTRAINT permissions_pkey PRIMARY KEY (id);
    END IF;
END;
$$;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conrelid = 'public.pricing_rules'::regclass AND conname = 'pricing_rules_name_unique'
    ) THEN
        ALTER TABLE ONLY public.pricing_rules ADD CONSTRAINT pricing_rules_name_unique UNIQUE (name);
    END IF;
END;
$$;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conrelid = 'public.pricing_rules'::regclass AND conname = 'pricing_rules_pkey'
    ) THEN
        ALTER TABLE ONLY public.pricing_rules ADD CONSTRAINT pricing_rules_pkey PRIMARY KEY (id);
    END IF;
END;
$$;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conrelid = 'public.product_stock'::regclass AND conname = 'product_stock_pkey'
    ) THEN
        ALTER TABLE ONLY public.product_stock ADD CONSTRAINT product_stock_pkey PRIMARY KEY (id);
    END IF;
END;
$$;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conrelid = 'public.product_suppliers'::regclass AND conname = 'product_suppliers_pkey'
    ) THEN
        ALTER TABLE ONLY public.product_suppliers ADD CONSTRAINT product_suppliers_pkey PRIMARY KEY (id);
    END IF;
END;
$$;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conrelid = 'public.product_suppliers'::regclass AND conname = 'product_suppliers_product_id_supplier_id_key'
    ) THEN
        ALTER TABLE ONLY public.product_suppliers ADD CONSTRAINT product_suppliers_product_id_supplier_id_key UNIQUE (product_id, supplier_id);
    END IF;
END;
$$;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conrelid = 'public.products'::regclass AND conname = 'products_pkey'
    ) THEN
        ALTER TABLE ONLY public.products ADD CONSTRAINT products_pkey PRIMARY KEY (id);
    END IF;
END;
$$;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conrelid = 'public.products'::regclass AND conname = 'products_sku_key'
    ) THEN
        ALTER TABLE ONLY public.products ADD CONSTRAINT products_sku_key UNIQUE (sku);
    END IF;
END;
$$;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conrelid = 'public.purchase_order_items'::regclass AND conname = 'purchase_order_items_pkey'
    ) THEN
        ALTER TABLE ONLY public.purchase_order_items ADD CONSTRAINT purchase_order_items_pkey PRIMARY KEY (id);
    END IF;
END;
$$;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conrelid = 'public.purchase_order_items'::regclass AND conname = 'purchase_order_items_purchase_order_id_product_id_key'
    ) THEN
        ALTER TABLE ONLY public.purchase_order_items ADD CONSTRAINT purchase_order_items_purchase_order_id_product_id_key UNIQUE (purchase_order_id, product_id);
    END IF;
END;
$$;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conrelid = 'public.purchase_orders'::regclass AND conname = 'purchase_orders_pkey'
    ) THEN
        ALTER TABLE ONLY public.purchase_orders ADD CONSTRAINT purchase_orders_pkey PRIMARY KEY (id);
    END IF;
END;
$$;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conrelid = 'public.purchase_orders'::regclass AND conname = 'purchase_orders_po_number_key'
    ) THEN
        ALTER TABLE ONLY public.purchase_orders ADD CONSTRAINT purchase_orders_po_number_key UNIQUE (po_number);
    END IF;
END;
$$;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conrelid = 'public.refresh_tokens'::regclass AND conname = 'refresh_tokens_pkey'
    ) THEN
        ALTER TABLE ONLY public.refresh_tokens ADD CONSTRAINT refresh_tokens_pkey PRIMARY KEY (id);
    END IF;
END;
$$;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conrelid = 'public.refresh_tokens'::regclass AND conname = 'refresh_tokens_token_hash_key'
    ) THEN
        ALTER TABLE ONLY public.refresh_tokens ADD CONSTRAINT refresh_tokens_token_hash_key UNIQUE (token_hash);
    END IF;
END;
$$;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conrelid = 'public.role_permissions'::regclass AND conname = 'role_permissions_pkey'
    ) THEN
        ALTER TABLE ONLY public.role_permissions ADD CONSTRAINT role_permissions_pkey PRIMARY KEY (role_id, permission_id);
    END IF;
END;
$$;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conrelid = 'public.roles'::regclass AND conname = 'roles_name_key'
    ) THEN
        ALTER TABLE ONLY public.roles ADD CONSTRAINT roles_name_key UNIQUE (name);
    END IF;
END;
$$;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conrelid = 'public.roles'::regclass AND conname = 'roles_pkey'
    ) THEN
        ALTER TABLE ONLY public.roles ADD CONSTRAINT roles_pkey PRIMARY KEY (id);
    END IF;
END;
$$;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conrelid = 'public.sale_items'::regclass AND conname = 'sale_items_pkey'
    ) THEN
        ALTER TABLE ONLY public.sale_items ADD CONSTRAINT sale_items_pkey PRIMARY KEY (id);
    END IF;
END;
$$;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conrelid = 'public.sale_payments'::regclass AND conname = 'sale_payments_pkey'
    ) THEN
        ALTER TABLE ONLY public.sale_payments ADD CONSTRAINT sale_payments_pkey PRIMARY KEY (id);
    END IF;
END;
$$;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conrelid = 'public.sales'::regclass AND conname = 'sales_invoice_number_key'
    ) THEN
        ALTER TABLE ONLY public.sales ADD CONSTRAINT sales_invoice_number_key UNIQUE (invoice_number);
    END IF;
END;
$$;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conrelid = 'public.sales'::regclass AND conname = 'sales_pkey'
    ) THEN
        ALTER TABLE ONLY public.sales ADD CONSTRAINT sales_pkey PRIMARY KEY (id);
    END IF;
END;
$$;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conrelid = 'public.schema_migrations'::regclass AND conname = 'schema_migrations_pkey'
    ) THEN
        ALTER TABLE ONLY public.schema_migrations ADD CONSTRAINT schema_migrations_pkey PRIMARY KEY (filename);
    END IF;
END;
$$;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conrelid = 'public.shifts'::regclass AND conname = 'shifts_pkey'
    ) THEN
        ALTER TABLE ONLY public.shifts ADD CONSTRAINT shifts_pkey PRIMARY KEY (id);
    END IF;
END;
$$;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conrelid = 'public.stock_opname_assignments'::regclass AND conname = 'stock_opname_assignments_pkey'
    ) THEN
        ALTER TABLE ONLY public.stock_opname_assignments ADD CONSTRAINT stock_opname_assignments_pkey PRIMARY KEY (id);
    END IF;
END;
$$;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conrelid = 'public.stock_opname_counts'::regclass AND conname = 'stock_opname_counts_pkey'
    ) THEN
        ALTER TABLE ONLY public.stock_opname_counts ADD CONSTRAINT stock_opname_counts_pkey PRIMARY KEY (id);
    END IF;
END;
$$;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conrelid = 'public.stock_opname_items'::regclass AND conname = 'stock_opname_items_pkey'
    ) THEN
        ALTER TABLE ONLY public.stock_opname_items ADD CONSTRAINT stock_opname_items_pkey PRIMARY KEY (id);
    END IF;
END;
$$;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conrelid = 'public.stock_opname_recount_requests'::regclass AND conname = 'stock_opname_recount_requests_pkey'
    ) THEN
        ALTER TABLE ONLY public.stock_opname_recount_requests ADD CONSTRAINT stock_opname_recount_requests_pkey PRIMARY KEY (id);
    END IF;
END;
$$;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conrelid = 'public.stock_opname_session_scopes'::regclass AND conname = 'stock_opname_session_scopes_pkey'
    ) THEN
        ALTER TABLE ONLY public.stock_opname_session_scopes ADD CONSTRAINT stock_opname_session_scopes_pkey PRIMARY KEY (id);
    END IF;
END;
$$;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conrelid = 'public.stock_opnames'::regclass AND conname = 'stock_opnames_pkey'
    ) THEN
        ALTER TABLE ONLY public.stock_opnames ADD CONSTRAINT stock_opnames_pkey PRIMARY KEY (id);
    END IF;
END;
$$;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conrelid = 'public.stock_opnames'::regclass AND conname = 'stock_opnames_session_number_key'
    ) THEN
        ALTER TABLE ONLY public.stock_opnames ADD CONSTRAINT stock_opnames_session_number_key UNIQUE (session_number);
    END IF;
END;
$$;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conrelid = 'public.storage_locations'::regclass AND conname = 'storage_locations_pkey'
    ) THEN
        ALTER TABLE ONLY public.storage_locations ADD CONSTRAINT storage_locations_pkey PRIMARY KEY (id);
    END IF;
END;
$$;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conrelid = 'public.stores'::regclass AND conname = 'stores_pkey'
    ) THEN
        ALTER TABLE ONLY public.stores ADD CONSTRAINT stores_pkey PRIMARY KEY (id);
    END IF;
END;
$$;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conrelid = 'public.suppliers'::regclass AND conname = 'suppliers_code_key'
    ) THEN
        ALTER TABLE ONLY public.suppliers ADD CONSTRAINT suppliers_code_key UNIQUE (code);
    END IF;
END;
$$;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conrelid = 'public.suppliers'::regclass AND conname = 'suppliers_pkey'
    ) THEN
        ALTER TABLE ONLY public.suppliers ADD CONSTRAINT suppliers_pkey PRIMARY KEY (id);
    END IF;
END;
$$;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conrelid = 'public.tax_classes'::regclass AND conname = 'tax_classes_name_key'
    ) THEN
        ALTER TABLE ONLY public.tax_classes ADD CONSTRAINT tax_classes_name_key UNIQUE (name);
    END IF;
END;
$$;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conrelid = 'public.tax_classes'::regclass AND conname = 'tax_classes_pkey'
    ) THEN
        ALTER TABLE ONLY public.tax_classes ADD CONSTRAINT tax_classes_pkey PRIMARY KEY (id);
    END IF;
END;
$$;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conrelid = 'public.units_of_measure'::regclass AND conname = 'units_of_measure_code_key'
    ) THEN
        ALTER TABLE ONLY public.units_of_measure ADD CONSTRAINT units_of_measure_code_key UNIQUE (code);
    END IF;
END;
$$;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conrelid = 'public.units_of_measure'::regclass AND conname = 'units_of_measure_pkey'
    ) THEN
        ALTER TABLE ONLY public.units_of_measure ADD CONSTRAINT units_of_measure_pkey PRIMARY KEY (id);
    END IF;
END;
$$;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conrelid = 'public.consignment_stock'::regclass AND conname = 'uq_consignment_stock_product'
    ) THEN
        ALTER TABLE ONLY public.consignment_stock ADD CONSTRAINT uq_consignment_stock_product UNIQUE (product_id);
    END IF;
END;
$$;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conrelid = 'public.consignment_terms'::regclass AND conname = 'uq_consignment_terms_arrangement_product'
    ) THEN
        ALTER TABLE ONLY public.consignment_terms ADD CONSTRAINT uq_consignment_terms_arrangement_product UNIQUE (arrangement_id, product_id);
    END IF;
END;
$$;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conrelid = 'public.product_stock'::regclass AND conname = 'uq_product_stock'
    ) THEN
        ALTER TABLE ONLY public.product_stock ADD CONSTRAINT uq_product_stock UNIQUE NULLS NOT DISTINCT (product_id, warehouse_id, store_id, location_id);
    END IF;
END;
$$;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conrelid = 'public.storage_locations'::regclass AND conname = 'uq_storage_locations_code'
    ) THEN
        ALTER TABLE ONLY public.storage_locations ADD CONSTRAINT uq_storage_locations_code UNIQUE (code);
    END IF;
END;
$$;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conrelid = 'public.users'::regclass AND conname = 'users_email_key'
    ) THEN
        ALTER TABLE ONLY public.users ADD CONSTRAINT users_email_key UNIQUE (email);
    END IF;
END;
$$;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conrelid = 'public.users'::regclass AND conname = 'users_pkey'
    ) THEN
        ALTER TABLE ONLY public.users ADD CONSTRAINT users_pkey PRIMARY KEY (id);
    END IF;
END;
$$;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conrelid = 'public.users'::regclass AND conname = 'users_username_key'
    ) THEN
        ALTER TABLE ONLY public.users ADD CONSTRAINT users_username_key UNIQUE (username);
    END IF;
END;
$$;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conrelid = 'public.warehouses'::regclass AND conname = 'warehouses_code_key'
    ) THEN
        ALTER TABLE ONLY public.warehouses ADD CONSTRAINT warehouses_code_key UNIQUE (code);
    END IF;
END;
$$;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conrelid = 'public.warehouses'::regclass AND conname = 'warehouses_pkey'
    ) THEN
        ALTER TABLE ONLY public.warehouses ADD CONSTRAINT warehouses_pkey PRIMARY KEY (id);
    END IF;
END;
$$;

-- ============================================================================
-- Indexes
-- ============================================================================

CREATE INDEX IF NOT EXISTS idx_audit_logs_action_ip_created ON public.audit_logs USING btree (action, ip_address, created_at);

CREATE INDEX IF NOT EXISTS idx_audit_logs_created ON public.audit_logs USING btree (created_at);

CREATE INDEX IF NOT EXISTS idx_audit_logs_created_id ON public.audit_logs USING btree (created_at DESC, id DESC);

CREATE INDEX IF NOT EXISTS idx_audit_logs_store ON public.audit_logs USING btree (store_id);

CREATE INDEX IF NOT EXISTS idx_audit_logs_store_created_id ON public.audit_logs USING btree (store_id, created_at DESC, id DESC) WHERE (store_id IS NOT NULL);

CREATE INDEX IF NOT EXISTS idx_audit_logs_user ON public.audit_logs USING btree (user_id);

CREATE INDEX IF NOT EXISTS idx_cart_items_session ON public.cart_items USING btree (cart_session_id);

CREATE INDEX IF NOT EXISTS idx_cart_sessions_cashier_status ON public.cart_sessions USING btree (cashier_id, status);

CREATE INDEX IF NOT EXISTS idx_cart_sessions_shift ON public.cart_sessions USING btree (shift_id);

CREATE INDEX IF NOT EXISTS idx_cash_movements_shift_id ON public.cash_movements USING btree (shift_id);

CREATE INDEX IF NOT EXISTS idx_cash_movements_shift_type ON public.cash_movements USING btree (shift_id, type);

CREATE INDEX IF NOT EXISTS idx_categories_slug ON public.categories USING btree (slug);

CREATE INDEX IF NOT EXISTS idx_consignment_arrangements_status ON public.consignment_arrangements USING btree (status);

CREATE INDEX IF NOT EXISTS idx_consignment_arrangements_supplier ON public.consignment_arrangements USING btree (supplier_id);

CREATE INDEX IF NOT EXISTS idx_consignment_pending_returns_open ON public.consignment_pending_returns USING btree (supplier_id, status);

CREATE INDEX IF NOT EXISTS idx_consignment_receipt_edits_edited_at ON public.consignment_receipt_edits USING btree (edited_at);

CREATE INDEX IF NOT EXISTS idx_consignment_receipt_edits_edited_by ON public.consignment_receipt_edits USING btree (edited_by);

CREATE INDEX IF NOT EXISTS idx_consignment_receipt_edits_receipt ON public.consignment_receipt_edits USING btree (receipt_id);

CREATE INDEX IF NOT EXISTS idx_consignment_sale_items_unsettled ON public.consignment_sale_items USING btree (supplier_id, settlement_id);

CREATE INDEX IF NOT EXISTS idx_consignment_settlement_items_settlement ON public.consignment_settlement_items USING btree (consignment_settlement_id);

CREATE INDEX IF NOT EXISTS idx_consignment_stock_supplier ON public.consignment_stock USING btree (supplier_id);

CREATE INDEX IF NOT EXISTS idx_consignment_terms_product ON public.consignment_terms USING btree (product_id);

CREATE INDEX IF NOT EXISTS idx_customer_groups_active ON public.customer_groups USING btree (is_active) WHERE (is_active = true);

CREATE INDEX IF NOT EXISTS idx_customer_groups_name ON public.customer_groups USING btree (name);

CREATE INDEX IF NOT EXISTS idx_customers_customer_group ON public.customers USING btree (customer_group_id);

CREATE INDEX IF NOT EXISTS idx_customers_is_active ON public.customers USING btree (is_active);

CREATE INDEX IF NOT EXISTS idx_customers_name_trgm ON public.customers USING gin (name public.gin_trgm_ops);

CREATE INDEX IF NOT EXISTS idx_customers_phone ON public.customers USING btree (phone);

CREATE INDEX IF NOT EXISTS idx_customers_store_id ON public.customers USING btree (store_id);

CREATE INDEX IF NOT EXISTS idx_dead_letter_events_created_at ON public.dead_letter_events USING btree (created_at);

CREATE INDEX IF NOT EXISTS idx_dead_letter_events_event_type ON public.dead_letter_events USING btree (event_type);

CREATE INDEX IF NOT EXISTS idx_goods_receipt_items_gr ON public.goods_receipt_items USING btree (goods_receipt_id);

CREATE INDEX IF NOT EXISTS idx_goods_receipts_po ON public.goods_receipts USING btree (purchase_order_id);

CREATE INDEX IF NOT EXISTS idx_ia_created ON public.inventory_adjustments USING btree (created_at DESC);

CREATE INDEX IF NOT EXISTS idx_ia_items_adj ON public.inventory_adjustment_items USING btree (adjustment_id);

CREATE INDEX IF NOT EXISTS idx_ia_items_product ON public.inventory_adjustment_items USING btree (product_id);

CREATE INDEX IF NOT EXISTS idx_ia_session ON public.inventory_adjustments USING btree (session_id);

CREATE INDEX IF NOT EXISTS idx_import_errors_job ON public.import_errors USING btree (import_job_id);

CREATE INDEX IF NOT EXISTS idx_import_jobs_module ON public.import_jobs USING btree (module);

CREATE INDEX IF NOT EXISTS idx_import_jobs_status ON public.import_jobs USING btree (status);

CREATE INDEX IF NOT EXISTS idx_import_jobs_user ON public.import_jobs USING btree (user_id);

CREATE INDEX IF NOT EXISTS idx_import_rows_job ON public.import_rows USING btree (import_job_id);

CREATE INDEX IF NOT EXISTS idx_inventory_movements_product ON public.inventory_movements USING btree (product_id);

CREATE UNIQUE INDEX IF NOT EXISTS idx_mv_daily_sales_date_store ON public.mv_daily_sales USING btree (sale_date, store_id);

CREATE UNIQUE INDEX IF NOT EXISTS idx_mv_dashboard_totals_store ON public.mv_dashboard_totals USING btree (store_id);

CREATE UNIQUE INDEX IF NOT EXISTS idx_mv_hourly_sales_hour_store ON public.mv_hourly_sales USING btree (sale_hour, store_id);

CREATE INDEX IF NOT EXISTS idx_payment_methods_code ON public.payment_methods USING btree (code);

CREATE INDEX IF NOT EXISTS idx_payment_methods_is_active ON public.payment_methods USING btree (is_active);

CREATE INDEX IF NOT EXISTS idx_pricing_rules_active ON public.pricing_rules USING btree (product_id, is_active) WHERE (is_active = true);

CREATE INDEX IF NOT EXISTS idx_pricing_rules_brand ON public.pricing_rules USING btree (brand_id) WHERE (brand_id IS NOT NULL);

CREATE INDEX IF NOT EXISTS idx_pricing_rules_category ON public.pricing_rules USING btree (category_id) WHERE (category_id IS NOT NULL);

CREATE INDEX IF NOT EXISTS idx_pricing_rules_effective ON public.pricing_rules USING btree (effective_from, effective_until) WHERE (is_active = true);

CREATE INDEX IF NOT EXISTS idx_pricing_rules_group ON public.pricing_rules USING btree (customer_group_id) WHERE (customer_group_id IS NOT NULL);

CREATE INDEX IF NOT EXISTS idx_pricing_rules_method ON public.pricing_rules USING btree (pricing_method);

CREATE INDEX IF NOT EXISTS idx_pricing_rules_product_id ON public.pricing_rules USING btree (product_id);

CREATE INDEX IF NOT EXISTS idx_pricing_rules_status ON public.pricing_rules USING btree (status);

CREATE INDEX IF NOT EXISTS idx_pricing_rules_store ON public.pricing_rules USING btree (store_id) WHERE (store_id IS NOT NULL);

CREATE INDEX IF NOT EXISTS idx_pricing_rules_type ON public.pricing_rules USING btree (pricing_type);

CREATE INDEX IF NOT EXISTS idx_product_stock_location_id ON public.product_stock USING btree (location_id);

CREATE INDEX IF NOT EXISTS idx_product_stock_product_id ON public.product_stock USING btree (product_id);

CREATE INDEX IF NOT EXISTS idx_product_stock_store_id ON public.product_stock USING btree (store_id);

CREATE INDEX IF NOT EXISTS idx_product_stock_warehouse_id ON public.product_stock USING btree (warehouse_id);

CREATE UNIQUE INDEX IF NOT EXISTS idx_product_suppliers_one_preferred ON public.product_suppliers USING btree (product_id) WHERE (is_preferred = true);

CREATE INDEX IF NOT EXISTS idx_product_suppliers_product ON public.product_suppliers USING btree (product_id);

CREATE INDEX IF NOT EXISTS idx_product_suppliers_supplier ON public.product_suppliers USING btree (supplier_id);

CREATE INDEX IF NOT EXISTS idx_products_barcode ON public.products USING btree (barcode);

CREATE INDEX IF NOT EXISTS idx_products_brand ON public.products USING btree (brand_id);

CREATE INDEX IF NOT EXISTS idx_products_category ON public.products USING btree (category_id);

CREATE INDEX IF NOT EXISTS idx_products_category_active ON public.products USING btree (category_id) WHERE (deleted_at IS NULL);

CREATE INDEX IF NOT EXISTS idx_products_name_trgm ON public.products USING gin (name public.gin_trgm_ops);

CREATE INDEX IF NOT EXISTS idx_products_ownership_type ON public.products USING btree (ownership_type);

CREATE INDEX IF NOT EXISTS idx_products_search_vector ON public.products USING gin (search_vector);

CREATE INDEX IF NOT EXISTS idx_products_sku ON public.products USING btree (sku);

CREATE INDEX IF NOT EXISTS idx_products_store ON public.products USING btree (store_id);

CREATE INDEX IF NOT EXISTS idx_products_tax_class ON public.products USING btree (tax_class_id);

CREATE UNIQUE INDEX IF NOT EXISTS idx_products_unique_active_barcode ON public.products USING btree (barcode) WHERE (deleted_at IS NULL);

CREATE INDEX IF NOT EXISTS idx_products_uom ON public.products USING btree (unit_of_measure_id);

CREATE INDEX IF NOT EXISTS idx_purchase_order_items_po ON public.purchase_order_items USING btree (purchase_order_id);

CREATE INDEX IF NOT EXISTS idx_purchase_orders_created_at ON public.purchase_orders USING btree (created_at DESC);

CREATE INDEX IF NOT EXISTS idx_purchase_orders_status ON public.purchase_orders USING btree (status);

CREATE INDEX IF NOT EXISTS idx_purchase_orders_status_store ON public.purchase_orders USING btree (status, store_id);

CREATE INDEX IF NOT EXISTS idx_purchase_orders_store ON public.purchase_orders USING btree (store_id);

CREATE INDEX IF NOT EXISTS idx_purchase_orders_supplier ON public.purchase_orders USING btree (supplier_id);

CREATE INDEX IF NOT EXISTS idx_sale_items_pricing_type ON public.sale_items USING btree (pricing_type);

CREATE INDEX IF NOT EXISTS idx_sale_items_product ON public.sale_items USING btree (product_id);

CREATE INDEX IF NOT EXISTS idx_sale_items_sale ON public.sale_items USING btree (sale_id);

CREATE INDEX IF NOT EXISTS idx_sale_items_sale_id ON public.sale_items USING btree (sale_id) INCLUDE (product_id, quantity, unit_price, subtotal);

CREATE INDEX IF NOT EXISTS idx_sale_payments_method ON public.sale_payments USING btree (payment_method_id);

CREATE INDEX IF NOT EXISTS idx_sale_payments_sale ON public.sale_payments USING btree (sale_id);

CREATE INDEX IF NOT EXISTS idx_sales_active_aggregations ON public.sales USING btree (created_at DESC) WHERE ((status)::text = 'completed'::text);

CREATE INDEX IF NOT EXISTS idx_sales_aggregation ON public.sales USING btree (store_id, created_at DESC, total_amount) INCLUDE (id, invoice_number, cashier_id, status);

CREATE INDEX IF NOT EXISTS idx_sales_cashier ON public.sales USING btree (cashier_id);

CREATE INDEX IF NOT EXISTS idx_sales_created ON public.sales USING btree (created_at);

CREATE INDEX IF NOT EXISTS idx_sales_created_id ON public.sales USING btree (created_at DESC, id DESC);

CREATE INDEX IF NOT EXISTS idx_sales_invoice_number_trgm ON public.sales USING gin (invoice_number public.gin_trgm_ops);

CREATE INDEX IF NOT EXISTS idx_sales_shift_id ON public.sales USING btree (shift_id);

CREATE INDEX IF NOT EXISTS idx_sales_shift_status ON public.sales USING btree (shift_id, status);

CREATE INDEX IF NOT EXISTS idx_sales_status_created_store ON public.sales USING btree (status, created_at, store_id) INCLUDE (total_amount);

CREATE INDEX IF NOT EXISTS idx_sales_store ON public.sales USING btree (store_id);

CREATE INDEX IF NOT EXISTS idx_shifts_opened_at ON public.shifts USING btree (opened_at);

CREATE INDEX IF NOT EXISTS idx_shifts_status ON public.shifts USING btree (status);

CREATE INDEX IF NOT EXISTS idx_shifts_store_id ON public.shifts USING btree (store_id);

CREATE INDEX IF NOT EXISTS idx_shifts_user_id ON public.shifts USING btree (user_id);

CREATE INDEX IF NOT EXISTS idx_so_items_opname_status ON public.stock_opname_items USING btree (stock_opname_id, status);

CREATE INDEX IF NOT EXISTS idx_so_recounts_opname ON public.stock_opname_recount_requests USING btree (stock_opname_id);

CREATE INDEX IF NOT EXISTS idx_so_scopes_opname ON public.stock_opname_session_scopes USING btree (stock_opname_id);

CREATE INDEX IF NOT EXISTS idx_so_scopes_type_id ON public.stock_opname_session_scopes USING btree (scope_type, scope_id);

CREATE INDEX IF NOT EXISTS idx_stock_opname_assignments_opname ON public.stock_opname_assignments USING btree (stock_opname_id);

CREATE INDEX IF NOT EXISTS idx_stock_opname_assignments_user ON public.stock_opname_assignments USING btree (user_id);

CREATE INDEX IF NOT EXISTS idx_stock_opname_counts_counted ON public.stock_opname_counts USING btree (counted_at);

CREATE INDEX IF NOT EXISTS idx_stock_opname_counts_item ON public.stock_opname_counts USING btree (stock_opname_item_id);

CREATE INDEX IF NOT EXISTS idx_stock_opname_created ON public.stock_opnames USING btree (created_at DESC);

CREATE INDEX IF NOT EXISTS idx_stock_opname_items_opname ON public.stock_opname_items USING btree (stock_opname_id);

CREATE INDEX IF NOT EXISTS idx_stock_opname_items_product ON public.stock_opname_items USING btree (product_id);

CREATE INDEX IF NOT EXISTS idx_stock_opname_items_status ON public.stock_opname_items USING btree (status);

CREATE INDEX IF NOT EXISTS idx_stock_opname_scope ON public.stock_opnames USING btree (scope_type, scope_id);

CREATE INDEX IF NOT EXISTS idx_stock_opname_status ON public.stock_opnames USING btree (status);

CREATE INDEX IF NOT EXISTS idx_stock_opname_status_created ON public.stock_opnames USING btree (status, created_at);

CREATE INDEX IF NOT EXISTS idx_stock_opnames_location_id ON public.stock_opnames USING btree (location_id);

CREATE INDEX IF NOT EXISTS idx_stock_opnames_store_id ON public.stock_opnames USING btree (store_id);

CREATE INDEX IF NOT EXISTS idx_storage_locations_active ON public.storage_locations USING btree (is_active);

CREATE INDEX IF NOT EXISTS idx_storage_locations_store_id ON public.storage_locations USING btree (store_id);

CREATE INDEX IF NOT EXISTS idx_storage_locations_warehouse_id ON public.storage_locations USING btree (warehouse_id);

CREATE INDEX IF NOT EXISTS idx_suppliers_code ON public.suppliers USING btree (code);

CREATE INDEX IF NOT EXISTS idx_users_reports_to ON public.users USING btree (reports_to);

CREATE INDEX IF NOT EXISTS idx_users_role ON public.users USING btree (role_id);

CREATE INDEX IF NOT EXISTS idx_users_store ON public.users USING btree (store_id);

CREATE UNIQUE INDEX IF NOT EXISTS stores_name_lower_key ON public.stores USING btree (lower((name)::text));

CREATE UNIQUE INDEX IF NOT EXISTS uq_cart_sessions_open_cashier ON public.cart_sessions USING btree (cashier_id) WHERE ((status)::text = 'open'::text);

CREATE UNIQUE INDEX IF NOT EXISTS uq_consignment_arrangement_active ON public.consignment_arrangements USING btree (supplier_id, store_id) WHERE ((status)::text = 'active'::text);

CREATE UNIQUE INDEX IF NOT EXISTS uq_open_shift_per_user ON public.shifts USING btree (user_id) WHERE ((status)::text = 'open'::text);

CREATE UNIQUE INDEX IF NOT EXISTS uq_stock_opname_assignment ON public.stock_opname_assignments USING btree (stock_opname_id, user_id, role);

-- ============================================================================
-- Triggers
-- ============================================================================

DROP TRIGGER IF EXISTS trg_audit_logs_immutable ON public.audit_logs;

CREATE TRIGGER trg_audit_logs_immutable BEFORE DELETE OR UPDATE ON public.audit_logs FOR EACH STATEMENT EXECUTE FUNCTION public.reject_audit_log_modification();

DROP TRIGGER IF EXISTS trg_consignment_receipt_edits_immutable ON public.consignment_receipt_edits;

CREATE TRIGGER trg_consignment_receipt_edits_immutable BEFORE DELETE OR UPDATE ON public.consignment_receipt_edits FOR EACH ROW EXECUTE FUNCTION public.prevent_consignment_receipt_edit_modification();

DROP TRIGGER IF EXISTS trg_products_search_vector ON public.products;

CREATE TRIGGER trg_products_search_vector BEFORE INSERT OR UPDATE OF name, sku, barcode ON public.products FOR EACH ROW EXECUTE FUNCTION public.products_search_vector_update();

-- ============================================================================
-- Constraints
-- ============================================================================

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conrelid = 'public.audit_logs'::regclass AND conname = 'audit_logs_store_id_fkey'
    ) THEN
        ALTER TABLE ONLY public.audit_logs ADD CONSTRAINT audit_logs_store_id_fkey FOREIGN KEY (store_id) REFERENCES public.stores(id) ON DELETE SET NULL;
    END IF;
END;
$$;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conrelid = 'public.audit_logs'::regclass AND conname = 'audit_logs_user_id_fkey'
    ) THEN
        ALTER TABLE ONLY public.audit_logs ADD CONSTRAINT audit_logs_user_id_fkey FOREIGN KEY (user_id) REFERENCES public.users(id) ON DELETE SET NULL;
    END IF;
END;
$$;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conrelid = 'public.cart_items'::regclass AND conname = 'cart_items_cart_session_id_fkey'
    ) THEN
        ALTER TABLE ONLY public.cart_items ADD CONSTRAINT cart_items_cart_session_id_fkey FOREIGN KEY (cart_session_id) REFERENCES public.cart_sessions(id) ON DELETE CASCADE;
    END IF;
END;
$$;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conrelid = 'public.cart_items'::regclass AND conname = 'cart_items_pricing_rule_id_fkey'
    ) THEN
        ALTER TABLE ONLY public.cart_items ADD CONSTRAINT cart_items_pricing_rule_id_fkey FOREIGN KEY (pricing_rule_id) REFERENCES public.pricing_rules(id) ON DELETE SET NULL;
    END IF;
END;
$$;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conrelid = 'public.cart_items'::regclass AND conname = 'cart_items_product_id_fkey'
    ) THEN
        ALTER TABLE ONLY public.cart_items ADD CONSTRAINT cart_items_product_id_fkey FOREIGN KEY (product_id) REFERENCES public.products(id) ON DELETE RESTRICT;
    END IF;
END;
$$;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conrelid = 'public.cart_items'::regclass AND conname = 'cart_items_tax_class_id_fkey'
    ) THEN
        ALTER TABLE ONLY public.cart_items ADD CONSTRAINT cart_items_tax_class_id_fkey FOREIGN KEY (tax_class_id) REFERENCES public.tax_classes(id) ON DELETE SET NULL;
    END IF;
END;
$$;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conrelid = 'public.cart_sessions'::regclass AND conname = 'cart_sessions_cashier_id_fkey'
    ) THEN
        ALTER TABLE ONLY public.cart_sessions ADD CONSTRAINT cart_sessions_cashier_id_fkey FOREIGN KEY (cashier_id) REFERENCES public.users(id) ON DELETE RESTRICT;
    END IF;
END;
$$;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conrelid = 'public.cart_sessions'::regclass AND conname = 'cart_sessions_customer_id_fkey'
    ) THEN
        ALTER TABLE ONLY public.cart_sessions ADD CONSTRAINT cart_sessions_customer_id_fkey FOREIGN KEY (customer_id) REFERENCES public.customers(id);
    END IF;
END;
$$;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conrelid = 'public.cart_sessions'::regclass AND conname = 'cart_sessions_shift_id_fkey'
    ) THEN
        ALTER TABLE ONLY public.cart_sessions ADD CONSTRAINT cart_sessions_shift_id_fkey FOREIGN KEY (shift_id) REFERENCES public.shifts(id);
    END IF;
END;
$$;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conrelid = 'public.cart_sessions'::regclass AND conname = 'cart_sessions_store_id_fkey'
    ) THEN
        ALTER TABLE ONLY public.cart_sessions ADD CONSTRAINT cart_sessions_store_id_fkey FOREIGN KEY (store_id) REFERENCES public.stores(id) ON DELETE SET NULL;
    END IF;
END;
$$;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conrelid = 'public.cash_movements'::regclass AND conname = 'cash_movements_shift_id_fkey'
    ) THEN
        ALTER TABLE ONLY public.cash_movements ADD CONSTRAINT cash_movements_shift_id_fkey FOREIGN KEY (shift_id) REFERENCES public.shifts(id) ON DELETE RESTRICT;
    END IF;
END;
$$;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conrelid = 'public.cash_movements'::regclass AND conname = 'cash_movements_user_id_fkey'
    ) THEN
        ALTER TABLE ONLY public.cash_movements ADD CONSTRAINT cash_movements_user_id_fkey FOREIGN KEY (user_id) REFERENCES public.users(id) ON DELETE RESTRICT;
    END IF;
END;
$$;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conrelid = 'public.categories'::regclass AND conname = 'categories_parent_id_fkey'
    ) THEN
        ALTER TABLE ONLY public.categories ADD CONSTRAINT categories_parent_id_fkey FOREIGN KEY (parent_id) REFERENCES public.categories(id) ON DELETE SET NULL;
    END IF;
END;
$$;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conrelid = 'public.consignment_arrangements'::regclass AND conname = 'consignment_arrangements_created_by_fkey'
    ) THEN
        ALTER TABLE ONLY public.consignment_arrangements ADD CONSTRAINT consignment_arrangements_created_by_fkey FOREIGN KEY (created_by) REFERENCES public.users(id);
    END IF;
END;
$$;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conrelid = 'public.consignment_arrangements'::regclass AND conname = 'consignment_arrangements_store_id_fkey'
    ) THEN
        ALTER TABLE ONLY public.consignment_arrangements ADD CONSTRAINT consignment_arrangements_store_id_fkey FOREIGN KEY (store_id) REFERENCES public.stores(id);
    END IF;
END;
$$;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conrelid = 'public.consignment_arrangements'::regclass AND conname = 'consignment_arrangements_supplier_id_fkey'
    ) THEN
        ALTER TABLE ONLY public.consignment_arrangements ADD CONSTRAINT consignment_arrangements_supplier_id_fkey FOREIGN KEY (supplier_id) REFERENCES public.suppliers(id);
    END IF;
END;
$$;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conrelid = 'public.consignment_payouts'::regclass AND conname = 'consignment_payouts_paid_by_fkey'
    ) THEN
        ALTER TABLE ONLY public.consignment_payouts ADD CONSTRAINT consignment_payouts_paid_by_fkey FOREIGN KEY (paid_by) REFERENCES public.users(id);
    END IF;
END;
$$;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conrelid = 'public.consignment_payouts'::regclass AND conname = 'consignment_payouts_payment_method_id_fkey'
    ) THEN
        ALTER TABLE ONLY public.consignment_payouts ADD CONSTRAINT consignment_payouts_payment_method_id_fkey FOREIGN KEY (payment_method_id) REFERENCES public.payment_methods(id);
    END IF;
END;
$$;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conrelid = 'public.consignment_payouts'::regclass AND conname = 'consignment_payouts_settlement_id_fkey'
    ) THEN
        ALTER TABLE ONLY public.consignment_payouts ADD CONSTRAINT consignment_payouts_settlement_id_fkey FOREIGN KEY (settlement_id) REFERENCES public.consignment_settlements(id);
    END IF;
END;
$$;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conrelid = 'public.consignment_pending_returns'::regclass AND conname = 'consignment_pending_returns_arrangement_id_fkey'
    ) THEN
        ALTER TABLE ONLY public.consignment_pending_returns ADD CONSTRAINT consignment_pending_returns_arrangement_id_fkey FOREIGN KEY (arrangement_id) REFERENCES public.consignment_arrangements(id);
    END IF;
END;
$$;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conrelid = 'public.consignment_pending_returns'::regclass AND conname = 'consignment_pending_returns_created_by_fkey'
    ) THEN
        ALTER TABLE ONLY public.consignment_pending_returns ADD CONSTRAINT consignment_pending_returns_created_by_fkey FOREIGN KEY (created_by) REFERENCES public.users(id);
    END IF;
END;
$$;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conrelid = 'public.consignment_pending_returns'::regclass AND conname = 'consignment_pending_returns_product_id_fkey'
    ) THEN
        ALTER TABLE ONLY public.consignment_pending_returns ADD CONSTRAINT consignment_pending_returns_product_id_fkey FOREIGN KEY (product_id) REFERENCES public.products(id);
    END IF;
END;
$$;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conrelid = 'public.consignment_pending_returns'::regclass AND conname = 'consignment_pending_returns_store_id_fkey'
    ) THEN
        ALTER TABLE ONLY public.consignment_pending_returns ADD CONSTRAINT consignment_pending_returns_store_id_fkey FOREIGN KEY (store_id) REFERENCES public.stores(id);
    END IF;
END;
$$;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conrelid = 'public.consignment_pending_returns'::regclass AND conname = 'consignment_pending_returns_supplier_id_fkey'
    ) THEN
        ALTER TABLE ONLY public.consignment_pending_returns ADD CONSTRAINT consignment_pending_returns_supplier_id_fkey FOREIGN KEY (supplier_id) REFERENCES public.suppliers(id);
    END IF;
END;
$$;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conrelid = 'public.consignment_receipt_edits'::regclass AND conname = 'consignment_receipt_edits_edited_by_fkey'
    ) THEN
        ALTER TABLE ONLY public.consignment_receipt_edits ADD CONSTRAINT consignment_receipt_edits_edited_by_fkey FOREIGN KEY (edited_by) REFERENCES public.users(id) ON DELETE SET NULL;
    END IF;
END;
$$;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conrelid = 'public.consignment_receipt_edits'::regclass AND conname = 'consignment_receipt_edits_receipt_id_fkey'
    ) THEN
        ALTER TABLE ONLY public.consignment_receipt_edits ADD CONSTRAINT consignment_receipt_edits_receipt_id_fkey FOREIGN KEY (receipt_id) REFERENCES public.consignment_receipts(id) ON DELETE CASCADE;
    END IF;
END;
$$;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conrelid = 'public.consignment_receipt_items'::regclass AND conname = 'consignment_receipt_items_product_id_fkey'
    ) THEN
        ALTER TABLE ONLY public.consignment_receipt_items ADD CONSTRAINT consignment_receipt_items_product_id_fkey FOREIGN KEY (product_id) REFERENCES public.products(id);
    END IF;
END;
$$;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conrelid = 'public.consignment_receipt_items'::regclass AND conname = 'consignment_receipt_items_receipt_id_fkey'
    ) THEN
        ALTER TABLE ONLY public.consignment_receipt_items ADD CONSTRAINT consignment_receipt_items_receipt_id_fkey FOREIGN KEY (consignment_receipt_id) REFERENCES public.consignment_receipts(id) ON DELETE CASCADE;
    END IF;
END;
$$;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conrelid = 'public.consignment_receipts'::regclass AND conname = 'consignment_receipts_arrangement_id_fkey'
    ) THEN
        ALTER TABLE ONLY public.consignment_receipts ADD CONSTRAINT consignment_receipts_arrangement_id_fkey FOREIGN KEY (arrangement_id) REFERENCES public.consignment_arrangements(id);
    END IF;
END;
$$;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conrelid = 'public.consignment_receipts'::regclass AND conname = 'consignment_receipts_received_by_fkey'
    ) THEN
        ALTER TABLE ONLY public.consignment_receipts ADD CONSTRAINT consignment_receipts_received_by_fkey FOREIGN KEY (received_by) REFERENCES public.users(id);
    END IF;
END;
$$;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conrelid = 'public.consignment_receipts'::regclass AND conname = 'consignment_receipts_store_id_fkey'
    ) THEN
        ALTER TABLE ONLY public.consignment_receipts ADD CONSTRAINT consignment_receipts_store_id_fkey FOREIGN KEY (store_id) REFERENCES public.stores(id);
    END IF;
END;
$$;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conrelid = 'public.consignment_receipts'::regclass AND conname = 'consignment_receipts_supplier_id_fkey'
    ) THEN
        ALTER TABLE ONLY public.consignment_receipts ADD CONSTRAINT consignment_receipts_supplier_id_fkey FOREIGN KEY (supplier_id) REFERENCES public.suppliers(id);
    END IF;
END;
$$;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conrelid = 'public.consignment_return_items'::regclass AND conname = 'consignment_return_items_pending_return_id_fkey'
    ) THEN
        ALTER TABLE ONLY public.consignment_return_items ADD CONSTRAINT consignment_return_items_pending_return_id_fkey FOREIGN KEY (pending_return_id) REFERENCES public.consignment_pending_returns(id) ON DELETE SET NULL;
    END IF;
END;
$$;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conrelid = 'public.consignment_return_items'::regclass AND conname = 'consignment_return_items_product_id_fkey'
    ) THEN
        ALTER TABLE ONLY public.consignment_return_items ADD CONSTRAINT consignment_return_items_product_id_fkey FOREIGN KEY (product_id) REFERENCES public.products(id);
    END IF;
END;
$$;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conrelid = 'public.consignment_return_items'::regclass AND conname = 'consignment_return_items_return_id_fkey'
    ) THEN
        ALTER TABLE ONLY public.consignment_return_items ADD CONSTRAINT consignment_return_items_return_id_fkey FOREIGN KEY (consignment_return_id) REFERENCES public.consignment_returns(id) ON DELETE CASCADE;
    END IF;
END;
$$;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conrelid = 'public.consignment_returns'::regclass AND conname = 'consignment_returns_arrangement_id_fkey'
    ) THEN
        ALTER TABLE ONLY public.consignment_returns ADD CONSTRAINT consignment_returns_arrangement_id_fkey FOREIGN KEY (arrangement_id) REFERENCES public.consignment_arrangements(id);
    END IF;
END;
$$;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conrelid = 'public.consignment_returns'::regclass AND conname = 'consignment_returns_returned_by_fkey'
    ) THEN
        ALTER TABLE ONLY public.consignment_returns ADD CONSTRAINT consignment_returns_returned_by_fkey FOREIGN KEY (returned_by) REFERENCES public.users(id);
    END IF;
END;
$$;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conrelid = 'public.consignment_returns'::regclass AND conname = 'consignment_returns_store_id_fkey'
    ) THEN
        ALTER TABLE ONLY public.consignment_returns ADD CONSTRAINT consignment_returns_store_id_fkey FOREIGN KEY (store_id) REFERENCES public.stores(id);
    END IF;
END;
$$;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conrelid = 'public.consignment_returns'::regclass AND conname = 'consignment_returns_supplier_id_fkey'
    ) THEN
        ALTER TABLE ONLY public.consignment_returns ADD CONSTRAINT consignment_returns_supplier_id_fkey FOREIGN KEY (supplier_id) REFERENCES public.suppliers(id);
    END IF;
END;
$$;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conrelid = 'public.consignment_sale_items'::regclass AND conname = 'consignment_sale_items_arrangement_id_fkey'
    ) THEN
        ALTER TABLE ONLY public.consignment_sale_items ADD CONSTRAINT consignment_sale_items_arrangement_id_fkey FOREIGN KEY (arrangement_id) REFERENCES public.consignment_arrangements(id);
    END IF;
END;
$$;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conrelid = 'public.consignment_sale_items'::regclass AND conname = 'consignment_sale_items_product_id_fkey'
    ) THEN
        ALTER TABLE ONLY public.consignment_sale_items ADD CONSTRAINT consignment_sale_items_product_id_fkey FOREIGN KEY (product_id) REFERENCES public.products(id);
    END IF;
END;
$$;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conrelid = 'public.consignment_sale_items'::regclass AND conname = 'consignment_sale_items_sale_id_fkey'
    ) THEN
        ALTER TABLE ONLY public.consignment_sale_items ADD CONSTRAINT consignment_sale_items_sale_id_fkey FOREIGN KEY (sale_id) REFERENCES public.sales(id) ON DELETE CASCADE;
    END IF;
END;
$$;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conrelid = 'public.consignment_sale_items'::regclass AND conname = 'consignment_sale_items_store_id_fkey'
    ) THEN
        ALTER TABLE ONLY public.consignment_sale_items ADD CONSTRAINT consignment_sale_items_store_id_fkey FOREIGN KEY (store_id) REFERENCES public.stores(id);
    END IF;
END;
$$;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conrelid = 'public.consignment_sale_items'::regclass AND conname = 'consignment_sale_items_supplier_id_fkey'
    ) THEN
        ALTER TABLE ONLY public.consignment_sale_items ADD CONSTRAINT consignment_sale_items_supplier_id_fkey FOREIGN KEY (supplier_id) REFERENCES public.suppliers(id);
    END IF;
END;
$$;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conrelid = 'public.consignment_settlement_items'::regclass AND conname = 'consignment_settlement_items_product_id_fkey'
    ) THEN
        ALTER TABLE ONLY public.consignment_settlement_items ADD CONSTRAINT consignment_settlement_items_product_id_fkey FOREIGN KEY (product_id) REFERENCES public.products(id);
    END IF;
END;
$$;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conrelid = 'public.consignment_settlement_items'::regclass AND conname = 'consignment_settlement_items_sale_item_id_fkey'
    ) THEN
        ALTER TABLE ONLY public.consignment_settlement_items ADD CONSTRAINT consignment_settlement_items_sale_item_id_fkey FOREIGN KEY (consignment_sale_item_id) REFERENCES public.consignment_sale_items(id);
    END IF;
END;
$$;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conrelid = 'public.consignment_settlement_items'::regclass AND conname = 'consignment_settlement_items_settlement_id_fkey'
    ) THEN
        ALTER TABLE ONLY public.consignment_settlement_items ADD CONSTRAINT consignment_settlement_items_settlement_id_fkey FOREIGN KEY (consignment_settlement_id) REFERENCES public.consignment_settlements(id) ON DELETE CASCADE;
    END IF;
END;
$$;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conrelid = 'public.consignment_settlements'::regclass AND conname = 'consignment_settlements_created_by_fkey'
    ) THEN
        ALTER TABLE ONLY public.consignment_settlements ADD CONSTRAINT consignment_settlements_created_by_fkey FOREIGN KEY (created_by) REFERENCES public.users(id);
    END IF;
END;
$$;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conrelid = 'public.consignment_settlements'::regclass AND conname = 'consignment_settlements_store_id_fkey'
    ) THEN
        ALTER TABLE ONLY public.consignment_settlements ADD CONSTRAINT consignment_settlements_store_id_fkey FOREIGN KEY (store_id) REFERENCES public.stores(id);
    END IF;
END;
$$;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conrelid = 'public.consignment_settlements'::regclass AND conname = 'consignment_settlements_supplier_id_fkey'
    ) THEN
        ALTER TABLE ONLY public.consignment_settlements ADD CONSTRAINT consignment_settlements_supplier_id_fkey FOREIGN KEY (supplier_id) REFERENCES public.suppliers(id);
    END IF;
END;
$$;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conrelid = 'public.consignment_stock'::regclass AND conname = 'consignment_stock_arrangement_id_fkey'
    ) THEN
        ALTER TABLE ONLY public.consignment_stock ADD CONSTRAINT consignment_stock_arrangement_id_fkey FOREIGN KEY (arrangement_id) REFERENCES public.consignment_arrangements(id);
    END IF;
END;
$$;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conrelid = 'public.consignment_stock'::regclass AND conname = 'consignment_stock_product_id_fkey'
    ) THEN
        ALTER TABLE ONLY public.consignment_stock ADD CONSTRAINT consignment_stock_product_id_fkey FOREIGN KEY (product_id) REFERENCES public.products(id) ON DELETE CASCADE;
    END IF;
END;
$$;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conrelid = 'public.consignment_stock'::regclass AND conname = 'consignment_stock_store_id_fkey'
    ) THEN
        ALTER TABLE ONLY public.consignment_stock ADD CONSTRAINT consignment_stock_store_id_fkey FOREIGN KEY (store_id) REFERENCES public.stores(id);
    END IF;
END;
$$;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conrelid = 'public.consignment_stock'::regclass AND conname = 'consignment_stock_supplier_id_fkey'
    ) THEN
        ALTER TABLE ONLY public.consignment_stock ADD CONSTRAINT consignment_stock_supplier_id_fkey FOREIGN KEY (supplier_id) REFERENCES public.suppliers(id);
    END IF;
END;
$$;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conrelid = 'public.consignment_terms'::regclass AND conname = 'consignment_terms_arrangement_id_fkey'
    ) THEN
        ALTER TABLE ONLY public.consignment_terms ADD CONSTRAINT consignment_terms_arrangement_id_fkey FOREIGN KEY (arrangement_id) REFERENCES public.consignment_arrangements(id) ON DELETE CASCADE;
    END IF;
END;
$$;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conrelid = 'public.consignment_terms'::regclass AND conname = 'consignment_terms_created_by_fkey'
    ) THEN
        ALTER TABLE ONLY public.consignment_terms ADD CONSTRAINT consignment_terms_created_by_fkey FOREIGN KEY (created_by) REFERENCES public.users(id);
    END IF;
END;
$$;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conrelid = 'public.consignment_terms'::regclass AND conname = 'consignment_terms_product_id_fkey'
    ) THEN
        ALTER TABLE ONLY public.consignment_terms ADD CONSTRAINT consignment_terms_product_id_fkey FOREIGN KEY (product_id) REFERENCES public.products(id);
    END IF;
END;
$$;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conrelid = 'public.customers'::regclass AND conname = 'customers_customer_group_id_fkey'
    ) THEN
        ALTER TABLE ONLY public.customers ADD CONSTRAINT customers_customer_group_id_fkey FOREIGN KEY (customer_group_id) REFERENCES public.customer_groups(id) ON DELETE SET NULL;
    END IF;
END;
$$;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conrelid = 'public.goods_receipt_items'::regclass AND conname = 'goods_receipt_items_goods_receipt_id_fkey'
    ) THEN
        ALTER TABLE ONLY public.goods_receipt_items ADD CONSTRAINT goods_receipt_items_goods_receipt_id_fkey FOREIGN KEY (goods_receipt_id) REFERENCES public.goods_receipts(id) ON DELETE CASCADE;
    END IF;
END;
$$;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conrelid = 'public.goods_receipt_items'::regclass AND conname = 'goods_receipt_items_product_id_fkey'
    ) THEN
        ALTER TABLE ONLY public.goods_receipt_items ADD CONSTRAINT goods_receipt_items_product_id_fkey FOREIGN KEY (product_id) REFERENCES public.products(id);
    END IF;
END;
$$;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conrelid = 'public.goods_receipt_items'::regclass AND conname = 'goods_receipt_items_purchase_order_item_id_fkey'
    ) THEN
        ALTER TABLE ONLY public.goods_receipt_items ADD CONSTRAINT goods_receipt_items_purchase_order_item_id_fkey FOREIGN KEY (purchase_order_item_id) REFERENCES public.purchase_order_items(id);
    END IF;
END;
$$;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conrelid = 'public.goods_receipts'::regclass AND conname = 'goods_receipts_purchase_order_id_fkey'
    ) THEN
        ALTER TABLE ONLY public.goods_receipts ADD CONSTRAINT goods_receipts_purchase_order_id_fkey FOREIGN KEY (purchase_order_id) REFERENCES public.purchase_orders(id);
    END IF;
END;
$$;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conrelid = 'public.import_errors'::regclass AND conname = 'import_errors_import_job_id_fkey'
    ) THEN
        ALTER TABLE ONLY public.import_errors ADD CONSTRAINT import_errors_import_job_id_fkey FOREIGN KEY (import_job_id) REFERENCES public.import_jobs(id) ON DELETE CASCADE;
    END IF;
END;
$$;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conrelid = 'public.import_jobs'::regclass AND conname = 'import_jobs_store_id_fkey'
    ) THEN
        ALTER TABLE ONLY public.import_jobs ADD CONSTRAINT import_jobs_store_id_fkey FOREIGN KEY (store_id) REFERENCES public.stores(id);
    END IF;
END;
$$;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conrelid = 'public.import_jobs'::regclass AND conname = 'import_jobs_user_id_fkey'
    ) THEN
        ALTER TABLE ONLY public.import_jobs ADD CONSTRAINT import_jobs_user_id_fkey FOREIGN KEY (user_id) REFERENCES public.users(id);
    END IF;
END;
$$;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conrelid = 'public.import_rows'::regclass AND conname = 'import_rows_import_job_id_fkey'
    ) THEN
        ALTER TABLE ONLY public.import_rows ADD CONSTRAINT import_rows_import_job_id_fkey FOREIGN KEY (import_job_id) REFERENCES public.import_jobs(id) ON DELETE CASCADE;
    END IF;
END;
$$;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conrelid = 'public.import_snapshots'::regclass AND conname = 'import_snapshots_import_job_id_fkey'
    ) THEN
        ALTER TABLE ONLY public.import_snapshots ADD CONSTRAINT import_snapshots_import_job_id_fkey FOREIGN KEY (import_job_id) REFERENCES public.import_jobs(id) ON DELETE CASCADE;
    END IF;
END;
$$;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conrelid = 'public.inventory_adjustment_items'::regclass AND conname = 'inventory_adjustment_items_adjustment_id_fkey'
    ) THEN
        ALTER TABLE ONLY public.inventory_adjustment_items ADD CONSTRAINT inventory_adjustment_items_adjustment_id_fkey FOREIGN KEY (adjustment_id) REFERENCES public.inventory_adjustments(id) ON DELETE CASCADE;
    END IF;
END;
$$;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conrelid = 'public.inventory_adjustment_items'::regclass AND conname = 'inventory_adjustment_items_product_id_fkey'
    ) THEN
        ALTER TABLE ONLY public.inventory_adjustment_items ADD CONSTRAINT inventory_adjustment_items_product_id_fkey FOREIGN KEY (product_id) REFERENCES public.products(id);
    END IF;
END;
$$;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conrelid = 'public.inventory_adjustment_items'::regclass AND conname = 'inventory_adjustment_items_store_id_fkey'
    ) THEN
        ALTER TABLE ONLY public.inventory_adjustment_items ADD CONSTRAINT inventory_adjustment_items_store_id_fkey FOREIGN KEY (store_id) REFERENCES public.stores(id);
    END IF;
END;
$$;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conrelid = 'public.inventory_adjustment_items'::regclass AND conname = 'inventory_adjustment_items_warehouse_id_fkey'
    ) THEN
        ALTER TABLE ONLY public.inventory_adjustment_items ADD CONSTRAINT inventory_adjustment_items_warehouse_id_fkey FOREIGN KEY (warehouse_id) REFERENCES public.warehouses(id);
    END IF;
END;
$$;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conrelid = 'public.inventory_adjustments'::regclass AND conname = 'inventory_adjustments_created_by_fkey'
    ) THEN
        ALTER TABLE ONLY public.inventory_adjustments ADD CONSTRAINT inventory_adjustments_created_by_fkey FOREIGN KEY (created_by) REFERENCES public.users(id);
    END IF;
END;
$$;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conrelid = 'public.inventory_adjustments'::regclass AND conname = 'inventory_adjustments_session_id_fkey'
    ) THEN
        ALTER TABLE ONLY public.inventory_adjustments ADD CONSTRAINT inventory_adjustments_session_id_fkey FOREIGN KEY (session_id) REFERENCES public.stock_opnames(id) ON DELETE RESTRICT;
    END IF;
END;
$$;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conrelid = 'public.inventory_movements'::regclass AND conname = 'inventory_movements_product_id_fkey'
    ) THEN
        ALTER TABLE ONLY public.inventory_movements ADD CONSTRAINT inventory_movements_product_id_fkey FOREIGN KEY (product_id) REFERENCES public.products(id) ON DELETE CASCADE;
    END IF;
END;
$$;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conrelid = 'public.inventory_movements'::regclass AND conname = 'inventory_movements_user_id_fkey'
    ) THEN
        ALTER TABLE ONLY public.inventory_movements ADD CONSTRAINT inventory_movements_user_id_fkey FOREIGN KEY (user_id) REFERENCES public.users(id) ON DELETE SET NULL;
    END IF;
END;
$$;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conrelid = 'public.pricing_rules'::regclass AND conname = 'pricing_rules_brand_id_fkey'
    ) THEN
        ALTER TABLE ONLY public.pricing_rules ADD CONSTRAINT pricing_rules_brand_id_fkey FOREIGN KEY (brand_id) REFERENCES public.brands(id) ON DELETE CASCADE;
    END IF;
END;
$$;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conrelid = 'public.pricing_rules'::regclass AND conname = 'pricing_rules_category_id_fkey'
    ) THEN
        ALTER TABLE ONLY public.pricing_rules ADD CONSTRAINT pricing_rules_category_id_fkey FOREIGN KEY (category_id) REFERENCES public.categories(id) ON DELETE CASCADE;
    END IF;
END;
$$;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conrelid = 'public.pricing_rules'::regclass AND conname = 'pricing_rules_customer_group_id_fkey'
    ) THEN
        ALTER TABLE ONLY public.pricing_rules ADD CONSTRAINT pricing_rules_customer_group_id_fkey FOREIGN KEY (customer_group_id) REFERENCES public.customer_groups(id) ON DELETE SET NULL;
    END IF;
END;
$$;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conrelid = 'public.pricing_rules'::regclass AND conname = 'pricing_rules_product_id_fkey'
    ) THEN
        ALTER TABLE ONLY public.pricing_rules ADD CONSTRAINT pricing_rules_product_id_fkey FOREIGN KEY (product_id) REFERENCES public.products(id) ON DELETE CASCADE;
    END IF;
END;
$$;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conrelid = 'public.pricing_rules'::regclass AND conname = 'pricing_rules_store_id_fkey'
    ) THEN
        ALTER TABLE ONLY public.pricing_rules ADD CONSTRAINT pricing_rules_store_id_fkey FOREIGN KEY (store_id) REFERENCES public.stores(id) ON DELETE SET NULL;
    END IF;
END;
$$;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conrelid = 'public.product_stock'::regclass AND conname = 'product_stock_location_id_fkey'
    ) THEN
        ALTER TABLE ONLY public.product_stock ADD CONSTRAINT product_stock_location_id_fkey FOREIGN KEY (location_id) REFERENCES public.storage_locations(id) ON DELETE SET NULL;
    END IF;
END;
$$;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conrelid = 'public.product_stock'::regclass AND conname = 'product_stock_product_id_fkey'
    ) THEN
        ALTER TABLE ONLY public.product_stock ADD CONSTRAINT product_stock_product_id_fkey FOREIGN KEY (product_id) REFERENCES public.products(id) ON DELETE CASCADE;
    END IF;
END;
$$;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conrelid = 'public.product_stock'::regclass AND conname = 'product_stock_store_id_fkey'
    ) THEN
        ALTER TABLE ONLY public.product_stock ADD CONSTRAINT product_stock_store_id_fkey FOREIGN KEY (store_id) REFERENCES public.stores(id) ON DELETE SET NULL;
    END IF;
END;
$$;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conrelid = 'public.product_stock'::regclass AND conname = 'product_stock_warehouse_id_fkey'
    ) THEN
        ALTER TABLE ONLY public.product_stock ADD CONSTRAINT product_stock_warehouse_id_fkey FOREIGN KEY (warehouse_id) REFERENCES public.warehouses(id) ON DELETE SET NULL;
    END IF;
END;
$$;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conrelid = 'public.product_suppliers'::regclass AND conname = 'product_suppliers_product_id_fkey'
    ) THEN
        ALTER TABLE ONLY public.product_suppliers ADD CONSTRAINT product_suppliers_product_id_fkey FOREIGN KEY (product_id) REFERENCES public.products(id) ON DELETE CASCADE;
    END IF;
END;
$$;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conrelid = 'public.product_suppliers'::regclass AND conname = 'product_suppliers_supplier_id_fkey'
    ) THEN
        ALTER TABLE ONLY public.product_suppliers ADD CONSTRAINT product_suppliers_supplier_id_fkey FOREIGN KEY (supplier_id) REFERENCES public.suppliers(id) ON DELETE CASCADE;
    END IF;
END;
$$;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conrelid = 'public.products'::regclass AND conname = 'products_brand_id_fkey'
    ) THEN
        ALTER TABLE ONLY public.products ADD CONSTRAINT products_brand_id_fkey FOREIGN KEY (brand_id) REFERENCES public.brands(id) ON DELETE SET NULL;
    END IF;
END;
$$;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conrelid = 'public.products'::regclass AND conname = 'products_category_id_fkey'
    ) THEN
        ALTER TABLE ONLY public.products ADD CONSTRAINT products_category_id_fkey FOREIGN KEY (category_id) REFERENCES public.categories(id) ON DELETE RESTRICT;
    END IF;
END;
$$;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conrelid = 'public.products'::regclass AND conname = 'products_store_id_fkey'
    ) THEN
        ALTER TABLE ONLY public.products ADD CONSTRAINT products_store_id_fkey FOREIGN KEY (store_id) REFERENCES public.stores(id) ON DELETE SET NULL;
    END IF;
END;
$$;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conrelid = 'public.products'::regclass AND conname = 'products_tax_class_id_fkey'
    ) THEN
        ALTER TABLE ONLY public.products ADD CONSTRAINT products_tax_class_id_fkey FOREIGN KEY (tax_class_id) REFERENCES public.tax_classes(id) ON DELETE SET NULL;
    END IF;
END;
$$;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conrelid = 'public.products'::regclass AND conname = 'products_unit_of_measure_id_fkey'
    ) THEN
        ALTER TABLE ONLY public.products ADD CONSTRAINT products_unit_of_measure_id_fkey FOREIGN KEY (unit_of_measure_id) REFERENCES public.units_of_measure(id) ON DELETE SET NULL;
    END IF;
END;
$$;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conrelid = 'public.purchase_order_items'::regclass AND conname = 'purchase_order_items_product_id_fkey'
    ) THEN
        ALTER TABLE ONLY public.purchase_order_items ADD CONSTRAINT purchase_order_items_product_id_fkey FOREIGN KEY (product_id) REFERENCES public.products(id);
    END IF;
END;
$$;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conrelid = 'public.purchase_order_items'::regclass AND conname = 'purchase_order_items_purchase_order_id_fkey'
    ) THEN
        ALTER TABLE ONLY public.purchase_order_items ADD CONSTRAINT purchase_order_items_purchase_order_id_fkey FOREIGN KEY (purchase_order_id) REFERENCES public.purchase_orders(id) ON DELETE CASCADE;
    END IF;
END;
$$;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conrelid = 'public.purchase_orders'::regclass AND conname = 'purchase_orders_supplier_id_fkey'
    ) THEN
        ALTER TABLE ONLY public.purchase_orders ADD CONSTRAINT purchase_orders_supplier_id_fkey FOREIGN KEY (supplier_id) REFERENCES public.suppliers(id);
    END IF;
END;
$$;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conrelid = 'public.refresh_tokens'::regclass AND conname = 'refresh_tokens_user_id_fkey'
    ) THEN
        ALTER TABLE ONLY public.refresh_tokens ADD CONSTRAINT refresh_tokens_user_id_fkey FOREIGN KEY (user_id) REFERENCES public.users(id) ON DELETE CASCADE;
    END IF;
END;
$$;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conrelid = 'public.role_permissions'::regclass AND conname = 'role_permissions_permission_id_fkey'
    ) THEN
        ALTER TABLE ONLY public.role_permissions ADD CONSTRAINT role_permissions_permission_id_fkey FOREIGN KEY (permission_id) REFERENCES public.permissions(id) ON DELETE CASCADE;
    END IF;
END;
$$;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conrelid = 'public.role_permissions'::regclass AND conname = 'role_permissions_role_id_fkey'
    ) THEN
        ALTER TABLE ONLY public.role_permissions ADD CONSTRAINT role_permissions_role_id_fkey FOREIGN KEY (role_id) REFERENCES public.roles(id) ON DELETE CASCADE;
    END IF;
END;
$$;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conrelid = 'public.sale_items'::regclass AND conname = 'sale_items_pricing_rule_id_fkey'
    ) THEN
        ALTER TABLE ONLY public.sale_items ADD CONSTRAINT sale_items_pricing_rule_id_fkey FOREIGN KEY (pricing_rule_id) REFERENCES public.pricing_rules(id) ON DELETE SET NULL;
    END IF;
END;
$$;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conrelid = 'public.sale_items'::regclass AND conname = 'sale_items_product_id_fkey'
    ) THEN
        ALTER TABLE ONLY public.sale_items ADD CONSTRAINT sale_items_product_id_fkey FOREIGN KEY (product_id) REFERENCES public.products(id) ON DELETE RESTRICT;
    END IF;
END;
$$;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conrelid = 'public.sale_items'::regclass AND conname = 'sale_items_sale_id_fkey'
    ) THEN
        ALTER TABLE ONLY public.sale_items ADD CONSTRAINT sale_items_sale_id_fkey FOREIGN KEY (sale_id) REFERENCES public.sales(id) ON DELETE CASCADE;
    END IF;
END;
$$;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conrelid = 'public.sale_items'::regclass AND conname = 'sale_items_tax_class_id_fkey'
    ) THEN
        ALTER TABLE ONLY public.sale_items ADD CONSTRAINT sale_items_tax_class_id_fkey FOREIGN KEY (tax_class_id) REFERENCES public.tax_classes(id) ON DELETE SET NULL;
    END IF;
END;
$$;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conrelid = 'public.sale_payments'::regclass AND conname = 'sale_payments_payment_method_id_fkey'
    ) THEN
        ALTER TABLE ONLY public.sale_payments ADD CONSTRAINT sale_payments_payment_method_id_fkey FOREIGN KEY (payment_method_id) REFERENCES public.payment_methods(id) ON DELETE RESTRICT;
    END IF;
END;
$$;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conrelid = 'public.sale_payments'::regclass AND conname = 'sale_payments_sale_id_fkey'
    ) THEN
        ALTER TABLE ONLY public.sale_payments ADD CONSTRAINT sale_payments_sale_id_fkey FOREIGN KEY (sale_id) REFERENCES public.sales(id) ON DELETE CASCADE;
    END IF;
END;
$$;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conrelid = 'public.sales'::regclass AND conname = 'sales_cashier_id_fkey'
    ) THEN
        ALTER TABLE ONLY public.sales ADD CONSTRAINT sales_cashier_id_fkey FOREIGN KEY (cashier_id) REFERENCES public.users(id) ON DELETE RESTRICT;
    END IF;
END;
$$;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conrelid = 'public.sales'::regclass AND conname = 'sales_customer_id_fkey'
    ) THEN
        ALTER TABLE ONLY public.sales ADD CONSTRAINT sales_customer_id_fkey FOREIGN KEY (customer_id) REFERENCES public.customers(id);
    END IF;
END;
$$;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conrelid = 'public.sales'::regclass AND conname = 'sales_shift_id_fkey'
    ) THEN
        ALTER TABLE ONLY public.sales ADD CONSTRAINT sales_shift_id_fkey FOREIGN KEY (shift_id) REFERENCES public.shifts(id);
    END IF;
END;
$$;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conrelid = 'public.sales'::regclass AND conname = 'sales_store_id_fkey'
    ) THEN
        ALTER TABLE ONLY public.sales ADD CONSTRAINT sales_store_id_fkey FOREIGN KEY (store_id) REFERENCES public.stores(id) ON DELETE SET NULL;
    END IF;
END;
$$;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conrelid = 'public.shifts'::regclass AND conname = 'shifts_reviewed_by_fkey'
    ) THEN
        ALTER TABLE ONLY public.shifts ADD CONSTRAINT shifts_reviewed_by_fkey FOREIGN KEY (reviewed_by) REFERENCES public.users(id);
    END IF;
END;
$$;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conrelid = 'public.shifts'::regclass AND conname = 'shifts_store_id_fkey'
    ) THEN
        ALTER TABLE ONLY public.shifts ADD CONSTRAINT shifts_store_id_fkey FOREIGN KEY (store_id) REFERENCES public.stores(id) ON DELETE SET NULL;
    END IF;
END;
$$;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conrelid = 'public.shifts'::regclass AND conname = 'shifts_user_id_fkey'
    ) THEN
        ALTER TABLE ONLY public.shifts ADD CONSTRAINT shifts_user_id_fkey FOREIGN KEY (user_id) REFERENCES public.users(id) ON DELETE RESTRICT;
    END IF;
END;
$$;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conrelid = 'public.stock_opname_assignments'::regclass AND conname = 'stock_opname_assignments_stock_opname_id_fkey'
    ) THEN
        ALTER TABLE ONLY public.stock_opname_assignments ADD CONSTRAINT stock_opname_assignments_stock_opname_id_fkey FOREIGN KEY (stock_opname_id) REFERENCES public.stock_opnames(id) ON DELETE CASCADE;
    END IF;
END;
$$;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conrelid = 'public.stock_opname_assignments'::regclass AND conname = 'stock_opname_assignments_user_id_fkey'
    ) THEN
        ALTER TABLE ONLY public.stock_opname_assignments ADD CONSTRAINT stock_opname_assignments_user_id_fkey FOREIGN KEY (user_id) REFERENCES public.users(id);
    END IF;
END;
$$;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conrelid = 'public.stock_opname_counts'::regclass AND conname = 'stock_opname_counts_counted_by_fkey'
    ) THEN
        ALTER TABLE ONLY public.stock_opname_counts ADD CONSTRAINT stock_opname_counts_counted_by_fkey FOREIGN KEY (counted_by) REFERENCES public.users(id);
    END IF;
END;
$$;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conrelid = 'public.stock_opname_counts'::regclass AND conname = 'stock_opname_counts_stock_opname_item_id_fkey'
    ) THEN
        ALTER TABLE ONLY public.stock_opname_counts ADD CONSTRAINT stock_opname_counts_stock_opname_item_id_fkey FOREIGN KEY (stock_opname_item_id) REFERENCES public.stock_opname_items(id) ON DELETE CASCADE;
    END IF;
END;
$$;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conrelid = 'public.stock_opname_items'::regclass AND conname = 'stock_opname_items_product_id_fkey'
    ) THEN
        ALTER TABLE ONLY public.stock_opname_items ADD CONSTRAINT stock_opname_items_product_id_fkey FOREIGN KEY (product_id) REFERENCES public.products(id);
    END IF;
END;
$$;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conrelid = 'public.stock_opname_items'::regclass AND conname = 'stock_opname_items_stock_opname_id_fkey'
    ) THEN
        ALTER TABLE ONLY public.stock_opname_items ADD CONSTRAINT stock_opname_items_stock_opname_id_fkey FOREIGN KEY (stock_opname_id) REFERENCES public.stock_opnames(id) ON DELETE CASCADE;
    END IF;
END;
$$;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conrelid = 'public.stock_opname_items'::regclass AND conname = 'stock_opname_items_store_id_fkey'
    ) THEN
        ALTER TABLE ONLY public.stock_opname_items ADD CONSTRAINT stock_opname_items_store_id_fkey FOREIGN KEY (store_id) REFERENCES public.stores(id);
    END IF;
END;
$$;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conrelid = 'public.stock_opname_items'::regclass AND conname = 'stock_opname_items_warehouse_id_fkey'
    ) THEN
        ALTER TABLE ONLY public.stock_opname_items ADD CONSTRAINT stock_opname_items_warehouse_id_fkey FOREIGN KEY (warehouse_id) REFERENCES public.warehouses(id);
    END IF;
END;
$$;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conrelid = 'public.stock_opname_recount_requests'::regclass AND conname = 'stock_opname_recount_requests_requested_by_fkey'
    ) THEN
        ALTER TABLE ONLY public.stock_opname_recount_requests ADD CONSTRAINT stock_opname_recount_requests_requested_by_fkey FOREIGN KEY (requested_by) REFERENCES public.users(id);
    END IF;
END;
$$;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conrelid = 'public.stock_opname_recount_requests'::regclass AND conname = 'stock_opname_recount_requests_stock_opname_id_fkey'
    ) THEN
        ALTER TABLE ONLY public.stock_opname_recount_requests ADD CONSTRAINT stock_opname_recount_requests_stock_opname_id_fkey FOREIGN KEY (stock_opname_id) REFERENCES public.stock_opnames(id) ON DELETE CASCADE;
    END IF;
END;
$$;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conrelid = 'public.stock_opname_session_scopes'::regclass AND conname = 'stock_opname_session_scopes_stock_opname_id_fkey'
    ) THEN
        ALTER TABLE ONLY public.stock_opname_session_scopes ADD CONSTRAINT stock_opname_session_scopes_stock_opname_id_fkey FOREIGN KEY (stock_opname_id) REFERENCES public.stock_opnames(id) ON DELETE CASCADE;
    END IF;
END;
$$;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conrelid = 'public.stock_opnames'::regclass AND conname = 'stock_opnames_closed_by_fkey'
    ) THEN
        ALTER TABLE ONLY public.stock_opnames ADD CONSTRAINT stock_opnames_closed_by_fkey FOREIGN KEY (closed_by) REFERENCES public.users(id);
    END IF;
END;
$$;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conrelid = 'public.stock_opnames'::regclass AND conname = 'stock_opnames_location_id_fkey'
    ) THEN
        ALTER TABLE ONLY public.stock_opnames ADD CONSTRAINT stock_opnames_location_id_fkey FOREIGN KEY (location_id) REFERENCES public.storage_locations(id) ON DELETE SET NULL;
    END IF;
END;
$$;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conrelid = 'public.stock_opnames'::regclass AND conname = 'stock_opnames_opened_by_fkey'
    ) THEN
        ALTER TABLE ONLY public.stock_opnames ADD CONSTRAINT stock_opnames_opened_by_fkey FOREIGN KEY (opened_by) REFERENCES public.users(id);
    END IF;
END;
$$;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conrelid = 'public.stock_opnames'::regclass AND conname = 'stock_opnames_posted_by_fkey'
    ) THEN
        ALTER TABLE ONLY public.stock_opnames ADD CONSTRAINT stock_opnames_posted_by_fkey FOREIGN KEY (posted_by) REFERENCES public.users(id);
    END IF;
END;
$$;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conrelid = 'public.stock_opnames'::regclass AND conname = 'stock_opnames_store_id_fkey'
    ) THEN
        ALTER TABLE ONLY public.stock_opnames ADD CONSTRAINT stock_opnames_store_id_fkey FOREIGN KEY (store_id) REFERENCES public.stores(id) ON DELETE SET NULL;
    END IF;
END;
$$;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conrelid = 'public.stock_opnames'::regclass AND conname = 'stock_opnames_verified_by_fkey'
    ) THEN
        ALTER TABLE ONLY public.stock_opnames ADD CONSTRAINT stock_opnames_verified_by_fkey FOREIGN KEY (verified_by) REFERENCES public.users(id);
    END IF;
END;
$$;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conrelid = 'public.storage_locations'::regclass AND conname = 'storage_locations_store_id_fkey'
    ) THEN
        ALTER TABLE ONLY public.storage_locations ADD CONSTRAINT storage_locations_store_id_fkey FOREIGN KEY (store_id) REFERENCES public.stores(id) ON DELETE SET NULL;
    END IF;
END;
$$;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conrelid = 'public.storage_locations'::regclass AND conname = 'storage_locations_warehouse_id_fkey'
    ) THEN
        ALTER TABLE ONLY public.storage_locations ADD CONSTRAINT storage_locations_warehouse_id_fkey FOREIGN KEY (warehouse_id) REFERENCES public.warehouses(id) ON DELETE SET NULL;
    END IF;
END;
$$;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conrelid = 'public.suppliers'::regclass AND conname = 'suppliers_store_id_fkey'
    ) THEN
        ALTER TABLE ONLY public.suppliers ADD CONSTRAINT suppliers_store_id_fkey FOREIGN KEY (store_id) REFERENCES public.stores(id);
    END IF;
END;
$$;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conrelid = 'public.users'::regclass AND conname = 'users_reports_to_fkey'
    ) THEN
        ALTER TABLE ONLY public.users ADD CONSTRAINT users_reports_to_fkey FOREIGN KEY (reports_to) REFERENCES public.users(id) ON DELETE SET NULL;
    END IF;
END;
$$;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conrelid = 'public.warehouses'::regclass AND conname = 'warehouses_store_id_fkey'
    ) THEN
        ALTER TABLE ONLY public.warehouses ADD CONSTRAINT warehouses_store_id_fkey FOREIGN KEY (store_id) REFERENCES public.stores(id) ON DELETE SET NULL;
    END IF;
END;
$$;

-- ============================================================
-- Reference data
-- ============================================================
-- The nine tables below carry permanent seed rows. Everything else in
-- a migrated database is transactional and must start empty.

-- Roles first: permissions and role_permissions resolve through them.

-- Roles (6). Role ids are stable because role_permissions is seeded by id.

INSERT INTO public.roles (id, name, description, is_system)
VALUES (1, 'superadmin', 'Super Administrator', TRUE)
ON CONFLICT DO NOTHING;
INSERT INTO public.roles (id, name, description, is_system)
VALUES (2, 'manager', 'Manajer Toko — pengelolaan toko secara penuh', TRUE)
ON CONFLICT DO NOTHING;
INSERT INTO public.roles (id, name, description, is_system)
VALUES (3, 'supervisor', 'Supervisor — pengawasan operasional harian', TRUE)
ON CONFLICT DO NOTHING;
INSERT INTO public.roles (id, name, description, is_system)
VALUES (4, 'cashier', 'Kasir', TRUE)
ON CONFLICT DO NOTHING;
INSERT INTO public.roles (id, name, description, is_system)
VALUES (5, 'inventory_staff', 'Staf Inventaris — pengelolaan stok dan opname', TRUE)
ON CONFLICT DO NOTHING;
INSERT INTO public.roles (id, name, description, is_system)
VALUES (6, 'finance', 'Keuangan — mencatat pembayaran supplier dan melihat laporan', TRUE)
ON CONFLICT DO NOTHING;

-- Permissions (86) matching 000_squash.sql.

INSERT INTO public.permissions (id, code, name, description)
VALUES (1, 'app_settings.update', 'Ubah Pengaturan Aplikasi', 'Bisa mengubah pengaturan global aplikasi')
ON CONFLICT DO NOTHING;
INSERT INTO public.permissions (id, code, name, description)
VALUES (2, 'app_settings.view', 'Lihat Pengaturan Aplikasi', 'Bisa melihat pengaturan global aplikasi')
ON CONFLICT DO NOTHING;
INSERT INTO public.permissions (id, code, name, description)
VALUES (3, 'audit.export', 'Ekspor Log Audit', 'Mengekspor seluruh log audit ke berkas (CSV/XLSX). Lebih sensitif daripada sekadar melihat daftar.')
ON CONFLICT DO NOTHING;
INSERT INTO public.permissions (id, code, name, description)
VALUES (4, 'audit.view', 'Lihat Log Audit', 'Bisa melihat riwayat log audit')
ON CONFLICT DO NOTHING;
INSERT INTO public.permissions (id, code, name, description)
VALUES (5, 'category.create', 'Tambah Kategori', 'Bisa menambah kategori')
ON CONFLICT DO NOTHING;
INSERT INTO public.permissions (id, code, name, description)
VALUES (6, 'category.delete', 'Hapus Kategori', 'Bisa menghapus kategori')
ON CONFLICT DO NOTHING;
INSERT INTO public.permissions (id, code, name, description)
VALUES (7, 'category.export', 'Export Kategori', 'Bisa mengexport data kategori')
ON CONFLICT DO NOTHING;
INSERT INTO public.permissions (id, code, name, description)
VALUES (8, 'category.import', 'Import Kategori', 'Bisa mengimport data kategori')
ON CONFLICT DO NOTHING;
INSERT INTO public.permissions (id, code, name, description)
VALUES (9, 'category.update', 'Edit Kategori', 'Bisa mengubah kategori')
ON CONFLICT DO NOTHING;
INSERT INTO public.permissions (id, code, name, description)
VALUES (10, 'category.view', 'Lihat Kategori', 'Bisa melihat kategori')
ON CONFLICT DO NOTHING;
INSERT INTO public.permissions (id, code, name, description)
VALUES (11, 'consignment.view', 'Lihat Konsinyasi', 'Bisa melihat arrangement, stock, dan settlement konsinyasi')
ON CONFLICT DO NOTHING;
INSERT INTO public.permissions (id, code, name, description)
VALUES (12, 'consignment.create', 'Buat Transaksi Konsinyasi', 'Bisa membuat arrangement, receipt, pending return, dan return')
ON CONFLICT DO NOTHING;
INSERT INTO public.permissions (id, code, name, description)
VALUES (13, 'consignment.update', 'Ubah Terms Konsinyasi', 'Bisa mengubah harga dan hak/potongan toko pada arrangement konsinyasi')
ON CONFLICT DO NOTHING;
INSERT INTO public.permissions (id, code, name, description)
VALUES (14, 'consignment.settle', 'Settlement Konsinyasi', 'Bisa membuat settlement konsinyasi')
ON CONFLICT DO NOTHING;
INSERT INTO public.permissions (id, code, name, description)
VALUES (15, 'consignment.pay', 'Pembayaran Konsinyasi', 'Bisa mencatat pembayaran kepada supplier konsinyasi')
ON CONFLICT DO NOTHING;
INSERT INTO public.permissions (id, code, name, description)
VALUES (16, 'customer.create', 'Create Customer', 'Add new customers')
ON CONFLICT DO NOTHING;
INSERT INTO public.permissions (id, code, name, description)
VALUES (17, 'customer.delete', 'Delete Customer', 'Deactivate/delete customers')
ON CONFLICT DO NOTHING;
INSERT INTO public.permissions (id, code, name, description)
VALUES (18, 'customer.export', 'Export Pelanggan', 'Bisa mengexport data pelanggan')
ON CONFLICT DO NOTHING;
INSERT INTO public.permissions (id, code, name, description)
VALUES (19, 'customer.import', 'Import Pelanggan', 'Bisa mengimport data pelanggan')
ON CONFLICT DO NOTHING;
INSERT INTO public.permissions (id, code, name, description)
VALUES (20, 'customer.update', 'Update Customer', 'Edit customer information')
ON CONFLICT DO NOTHING;
INSERT INTO public.permissions (id, code, name, description)
VALUES (21, 'customer.view', 'View Customers', 'View customer list and details')
ON CONFLICT DO NOTHING;
INSERT INTO public.permissions (id, code, name, description)
VALUES (22, 'customer_group.create', 'Buat Data Customer Group', NULL)
ON CONFLICT DO NOTHING;
INSERT INTO public.permissions (id, code, name, description)
VALUES (23, 'customer_group.delete', 'Hapus Data Customer Group', NULL)
ON CONFLICT DO NOTHING;
INSERT INTO public.permissions (id, code, name, description)
VALUES (24, 'customer_group.update', 'Edit Data Customer Group', NULL)
ON CONFLICT DO NOTHING;
INSERT INTO public.permissions (id, code, name, description)
VALUES (25, 'customer_group.view', 'Lihat Data Customer Group', NULL)
ON CONFLICT DO NOTHING;
INSERT INTO public.permissions (id, code, name, description)
VALUES (26, 'dashboard.view', 'Lihat Dashboard', 'Bisa melihat dashboard utama')
ON CONFLICT DO NOTHING;
INSERT INTO public.permissions (id, code, name, description)
VALUES (27, 'inventory.adjust', 'Adjust Inventory', 'Manual stock adjustment')
ON CONFLICT DO NOTHING;
INSERT INTO public.permissions (id, code, name, description)
VALUES (28, 'pricing.create', 'Tambah Aturan Harga', 'Bisa menambah aturan harga baru')
ON CONFLICT DO NOTHING;
INSERT INTO public.permissions (id, code, name, description)
VALUES (29, 'pricing.delete', 'Hapus Aturan Harga', 'Bisa menghapus aturan harga')
ON CONFLICT DO NOTHING;
INSERT INTO public.permissions (id, code, name, description)
VALUES (30, 'pricing.update', 'Edit Aturan Harga', 'Bisa mengubah aturan harga')
ON CONFLICT DO NOTHING;
INSERT INTO public.permissions (id, code, name, description)
VALUES (31, 'pricing.view', 'Lihat Aturan Harga', 'Bisa melihat daftar aturan harga')
ON CONFLICT DO NOTHING;
INSERT INTO public.permissions (id, code, name, description)
VALUES (32, 'product.cost.view', 'Product Cost View', 'View sensitive cost data (cost, margin, purchase price, markup, profit)')
ON CONFLICT DO NOTHING;
INSERT INTO public.permissions (id, code, name, description)
VALUES (33, 'product.create', 'Tambah Produk', 'Bisa menambah produk baru')
ON CONFLICT DO NOTHING;
INSERT INTO public.permissions (id, code, name, description)
VALUES (34, 'product.delete', 'Hapus Produk', 'Bisa menghapus produk')
ON CONFLICT DO NOTHING;
INSERT INTO public.permissions (id, code, name, description)
VALUES (35, 'product.export', 'Export Produk', 'Bisa mengexport data produk')
ON CONFLICT DO NOTHING;
INSERT INTO public.permissions (id, code, name, description)
VALUES (36, 'product.history.view', 'Product History View', 'View product entity history (audit trail: created & updated timestamps)')
ON CONFLICT DO NOTHING;
INSERT INTO public.permissions (id, code, name, description)
VALUES (37, 'product.import', 'Import Produk', 'Bisa mengimport data produk')
ON CONFLICT DO NOTHING;
INSERT INTO public.permissions (id, code, name, description)
VALUES (38, 'product.update', 'Edit Produk', 'Bisa mengubah data produk')
ON CONFLICT DO NOTHING;
INSERT INTO public.permissions (id, code, name, description)
VALUES (39, 'product.view', 'Lihat Produk', 'Bisa melihat daftar produk')
ON CONFLICT DO NOTHING;
INSERT INTO public.permissions (id, code, name, description)
VALUES (40, 'purchase_order.cancel', 'Batalkan Purchase Order', 'Bisa membatalkan purchase order (draft/confirmed)')
ON CONFLICT DO NOTHING;
INSERT INTO public.permissions (id, code, name, description)
VALUES (41, 'purchase_order.confirm', 'Konfirmasi Purchase Order', 'Bisa mengkonfirmasi purchase order')
ON CONFLICT DO NOTHING;
INSERT INTO public.permissions (id, code, name, description)
VALUES (42, 'purchase_order.create', 'Buat Purchase Order', 'Bisa membuat purchase order baru')
ON CONFLICT DO NOTHING;
INSERT INTO public.permissions (id, code, name, description)
VALUES (43, 'purchase_order.delete', 'Hapus Purchase Order', 'Bisa menghapus purchase order')
ON CONFLICT DO NOTHING;
INSERT INTO public.permissions (id, code, name, description)
VALUES (44, 'purchase_order.receive', 'Terima Barang', 'Bisa membuat goods receipt untuk purchase order')
ON CONFLICT DO NOTHING;
INSERT INTO public.permissions (id, code, name, description)
VALUES (45, 'purchase_order.update', 'Edit Purchase Order', 'Bisa mengubah purchase order')
ON CONFLICT DO NOTHING;
INSERT INTO public.permissions (id, code, name, description)
VALUES (46, 'purchase_order.view', 'Lihat Purchase Order', 'Bisa melihat daftar dan detail purchase order')
ON CONFLICT DO NOTHING;
INSERT INTO public.permissions (id, code, name, description)
VALUES (47, 'receipt.print', 'Cetak Struk', 'Mencetak ulang struk penjualan, termasuk transaksi kasir lain.')
ON CONFLICT DO NOTHING;
INSERT INTO public.permissions (id, code, name, description)
VALUES (48, 'report.view', 'Lihat Laporan', 'Bisa melihat laporan keuangan & stok')
ON CONFLICT DO NOTHING;
INSERT INTO public.permissions (id, code, name, description)
VALUES (49, 'role.create', 'Tambah Role', 'Bisa menambah role baru')
ON CONFLICT DO NOTHING;
INSERT INTO public.permissions (id, code, name, description)
VALUES (50, 'role.delete', 'Hapus Role', 'Bisa menghapus role')
ON CONFLICT DO NOTHING;
INSERT INTO public.permissions (id, code, name, description)
VALUES (51, 'role.update', 'Edit Role', 'Bisa mengubah role')
ON CONFLICT DO NOTHING;
INSERT INTO public.permissions (id, code, name, description)
VALUES (52, 'role.view', 'Lihat Role', 'Bisa melihat daftar role')
ON CONFLICT DO NOTHING;
INSERT INTO public.permissions (id, code, name, description)
VALUES (53, 'sale.create', 'Buat Penjualan', 'Bisa membuat transaksi penjualan')
ON CONFLICT DO NOTHING;
INSERT INTO public.permissions (id, code, name, description)
VALUES (54, 'sale.detail', 'Cari Detail Penjualan Lintas Kasir', 'Melihat rincian item transaksi kasir lain (tanpa harga pokok/margin) untuk cetak ulang struk.')
ON CONFLICT DO NOTHING;
INSERT INTO public.permissions (id, code, name, description)
VALUES (55, 'sale.lookup', 'Cari Penjualan Lintas Kasir', 'Cari transaksi kasir lain secara ringkas (tanpa detail item/biaya/pelanggan)')
ON CONFLICT DO NOTHING;
INSERT INTO public.permissions (id, code, name, description)
VALUES (56, 'sale.park', 'Parkir Penjualan', 'Bisa menyimpan sementara, mengambil kembali, dan membatalkan penjualan yang diparkir')
ON CONFLICT DO NOTHING;
INSERT INTO public.permissions (id, code, name, description)
VALUES (57, 'sale.view', 'Lihat Penjualan', 'Bisa melihat daftar penjualan')
ON CONFLICT DO NOTHING;
INSERT INTO public.permissions (id, code, name, description)
VALUES (58, 'shift.audit', 'Audit shift', 'Audit fisik cash shift')
ON CONFLICT DO NOTHING;
INSERT INTO public.permissions (id, code, name, description)
VALUES (59, 'shift.cash_movement', 'Shift Cash Movement', 'Record cash drop / paid in / paid out')
ON CONFLICT DO NOTHING;
INSERT INTO public.permissions (id, code, name, description)
VALUES (60, 'shift.create', 'Kelola shift', 'Buka dan tutup shift')
ON CONFLICT DO NOTHING;
INSERT INTO public.permissions (id, code, name, description)
VALUES (61, 'shift.review', 'Review shift', 'Review dan setujui selisih shift')
ON CONFLICT DO NOTHING;
INSERT INTO public.permissions (id, code, name, description)
VALUES (62, 'shift.view', 'Baca shift', 'Lihat daftar dan detail shift')
ON CONFLICT DO NOTHING;
INSERT INTO public.permissions (id, code, name, description)
VALUES (63, 'stock_opname.assign', 'Atur Petugas Stock Opname', 'Bisa menugaskan counter dan supervisor')
ON CONFLICT DO NOTHING;
INSERT INTO public.permissions (id, code, name, description)
VALUES (64, 'stock_opname.cancel', 'Batalkan Stock Opname', 'Bisa membatalkan sesi stock opname')
ON CONFLICT DO NOTHING;
INSERT INTO public.permissions (id, code, name, description)
VALUES (65, 'stock_opname.close', 'Tutup Stock Opname', 'Bisa menutup sesi stock opname')
ON CONFLICT DO NOTHING;
INSERT INTO public.permissions (id, code, name, description)
VALUES (66, 'stock_opname.count', 'Hitung Stok Fisik', 'Bisa melakukan penghitungan stok fisik')
ON CONFLICT DO NOTHING;
INSERT INTO public.permissions (id, code, name, description)
VALUES (67, 'stock_opname.create', 'Buat Stock Opname', 'Bisa membuat sesi stock opname baru')
ON CONFLICT DO NOTHING;
INSERT INTO public.permissions (id, code, name, description)
VALUES (68, 'stock_opname.export', 'Ekspor Stock Opname', 'Bisa mengekspor laporan stock opname')
ON CONFLICT DO NOTHING;
INSERT INTO public.permissions (id, code, name, description)
VALUES (69, 'stock_opname.post', 'Posting Stock Opname', 'Bisa memposting penyesuaian stok')
ON CONFLICT DO NOTHING;
INSERT INTO public.permissions (id, code, name, description)
VALUES (70, 'stock_opname.recount', 'Minta Hitung Ulang', 'Bisa meminta penghitungan ulang')
ON CONFLICT DO NOTHING;
INSERT INTO public.permissions (id, code, name, description)
VALUES (71, 'stock_opname.report', 'Laporan Stock Opname', 'Bisa melihat laporan stock opname')
ON CONFLICT DO NOTHING;
INSERT INTO public.permissions (id, code, name, description)
VALUES (72, 'stock_opname.submit', 'Submit Stock Opname', 'Bisa mengirim hasil penghitungan')
ON CONFLICT DO NOTHING;
INSERT INTO public.permissions (id, code, name, description)
VALUES (73, 'stock_opname.verify', 'Verifikasi Stock Opname', 'Bisa memverifikasi hasil stock opname')
ON CONFLICT DO NOTHING;
INSERT INTO public.permissions (id, code, name, description)
VALUES (74, 'stock_opname.view', 'Lihat Stock Opname', 'Bisa melihat daftar dan detail stock opname')
ON CONFLICT DO NOTHING;
INSERT INTO public.permissions (id, code, name, description)
VALUES (75, 'storage_location.create', 'Buat Lokasi Penyimpanan', 'Bisa membuat lokasi penyimpanan baru')
ON CONFLICT DO NOTHING;
INSERT INTO public.permissions (id, code, name, description)
VALUES (76, 'storage_location.delete', 'Hapus Lokasi Penyimpanan', 'Bisa menghapus lokasi penyimpanan')
ON CONFLICT DO NOTHING;
INSERT INTO public.permissions (id, code, name, description)
VALUES (77, 'storage_location.update', 'Ubah Lokasi Penyimpanan', 'Bisa mengubah lokasi penyimpanan')
ON CONFLICT DO NOTHING;
INSERT INTO public.permissions (id, code, name, description)
VALUES (78, 'storage_location.view', 'Lihat Lokasi Penyimpanan', 'Bisa melihat daftar dan detail lokasi penyimpanan')
ON CONFLICT DO NOTHING;
INSERT INTO public.permissions (id, code, name, description)
VALUES (79, 'store.create', 'Buat Data Toko', NULL)
ON CONFLICT DO NOTHING;
INSERT INTO public.permissions (id, code, name, description)
VALUES (80, 'store.delete', 'Hapus Data Toko', NULL)
ON CONFLICT DO NOTHING;
INSERT INTO public.permissions (id, code, name, description)
VALUES (81, 'store.update', 'Edit Data Toko', NULL)
ON CONFLICT DO NOTHING;
INSERT INTO public.permissions (id, code, name, description)
VALUES (82, 'store.view', 'Lihat Data Toko', NULL)
ON CONFLICT DO NOTHING;
INSERT INTO public.permissions (id, code, name, description)
VALUES (83, 'user.create', 'Tambah Pengguna', 'Bisa menambah pengguna baru')
ON CONFLICT DO NOTHING;
INSERT INTO public.permissions (id, code, name, description)
VALUES (84, 'user.delete', 'Hapus Pengguna', 'Bisa menghapus pengguna')
ON CONFLICT DO NOTHING;
INSERT INTO public.permissions (id, code, name, description)
VALUES (85, 'user.update', 'Edit Pengguna', 'Bisa mengubah data pengguna')
ON CONFLICT DO NOTHING;
INSERT INTO public.permissions (id, code, name, description)
VALUES (86, 'user.view', 'Lihat Pengguna', 'Bisa melihat daftar pengguna')
ON CONFLICT DO NOTHING;

-- Tender types. sort_order drives the checkout button order.

INSERT INTO public.payment_methods (id, code, name, is_active, requires_reference, sort_order)
VALUES (1, 'CASH', 'Cash', TRUE, FALSE, 1)
ON CONFLICT DO NOTHING;
INSERT INTO public.payment_methods (id, code, name, is_active, requires_reference, sort_order)
VALUES (2, 'CARD', 'Card', TRUE, TRUE, 2)
ON CONFLICT DO NOTHING;
INSERT INTO public.payment_methods (id, code, name, is_active, requires_reference, sort_order)
VALUES (3, 'E_WALLET', 'E-Wallet', TRUE, TRUE, 3)
ON CONFLICT DO NOTHING;
INSERT INTO public.payment_methods (id, code, name, is_active, requires_reference, sort_order)
VALUES (4, 'TRANSFER', 'Transfer', TRUE, TRUE, 4)
ON CONFLICT DO NOTHING;
INSERT INTO public.payment_methods (id, code, name, is_active, requires_reference, sort_order)
VALUES (5, 'QRIS', 'QRIS', TRUE, FALSE, 5)
ON CONFLICT DO NOTHING;

-- Customer segments.

INSERT INTO public.customer_groups (id, name, description, is_active, color)
VALUES (1, 'Walk-in', 'Pelanggan umum tanpa kartu member', TRUE, '#636E72')
ON CONFLICT DO NOTHING;
INSERT INTO public.customer_groups (id, name, description, is_active, color)
VALUES (2, 'Member', 'Pelanggan terdaftar dengan kartu member', TRUE, '#00B894')
ON CONFLICT DO NOTHING;
INSERT INTO public.customer_groups (id, name, description, is_active, color)
VALUES (3, 'VIP', 'Pelanggan prioritas dengan harga khusus', TRUE, '#6C5CE7')
ON CONFLICT DO NOTHING;

-- Default store. id = 1 is relied on by the seeder and by tests that hard-code StoreID: 1.

INSERT INTO public.stores (id, name, address, phone, is_active)
VALUES (1, 'Default Store', 'Alamat toko default', '0000000000', TRUE)
ON CONFLICT DO NOTHING;

-- Walk-in customer. id = 1 is the implicit no-customer row used by checkout.

INSERT INTO public.customers (id, name, phone, email, address, tax_id, loyalty_points, total_spent, note, is_active, is_walk_in, store_id, customer_group_id)
VALUES (1, 'Pelanggan Umum / Walk-in', '0000000000', 'walk-in@retail-pos.local', NULL, NULL, 0, 0, NULL, TRUE, TRUE, 1, NULL)
ON CONFLICT DO NOTHING;

-- Application settings (7). 'key' is a reserved word and stays quoted.
-- 'key' is a SQL reserved word and must stay quoted.

INSERT INTO public.app_settings ("key", "value")
VALUES ('receipt_footer', 'Terima kasih atas kunjungan Anda!')
ON CONFLICT DO NOTHING;
INSERT INTO public.app_settings ("key", "value")
VALUES ('receipt_header', '')
ON CONFLICT DO NOTHING;
INSERT INTO public.app_settings ("key", "value")
VALUES ('shift_auto_close_hours', '24')
ON CONFLICT DO NOTHING;
INSERT INTO public.app_settings ("key", "value")
VALUES ('shift_blind_close', 'false')
ON CONFLICT DO NOTHING;
INSERT INTO public.app_settings ("key", "value")
VALUES ('shift_discrepancy_threshold', '50000')
ON CONFLICT DO NOTHING;
INSERT INTO public.app_settings ("key", "value")
VALUES ('store_jargon', 'Management System')
ON CONFLICT DO NOTHING;
INSERT INTO public.app_settings ("key", "value")
VALUES ('store_name', 'RetailPOS')
ON CONFLICT DO NOTHING;

-- The six system accounts seeded by 000_squash.sql. must_change_password comes from 052_first_install_hardening.sql and is set only for these accounts because their hash is generated here.
-- password_hash is regenerated per install with a random salt;
-- must_change_password is TRUE only for these six accounts,
-- guarded in 052_first_install_hardening.sql by
--   password_hash = crypt('admin123', password_hash)
--   which is true exactly for hashes made this way.

INSERT INTO public.users (id, username, email, password_hash, role_id, store_id, is_active, reports_to, language, theme, must_change_password)
VALUES (1, 'superadmin', 'superadmin@retailpos.local', crypt('admin123', gen_salt('bf', 14)), 1, NULL, TRUE, NULL, 'id', 'light', TRUE)
ON CONFLICT DO NOTHING;
INSERT INTO public.users (id, username, email, password_hash, role_id, store_id, is_active, reports_to, language, theme, must_change_password)
VALUES (2, 'manager', 'manager@retailpos.local', crypt('admin123', gen_salt('bf', 14)), 2, 1, TRUE, NULL, 'id', 'light', TRUE)
ON CONFLICT DO NOTHING;
INSERT INTO public.users (id, username, email, password_hash, role_id, store_id, is_active, reports_to, language, theme, must_change_password)
VALUES (3, 'supervisor', 'supervisor@retailpos.local', crypt('admin123', gen_salt('bf', 14)), 3, 1, TRUE, NULL, 'id', 'light', TRUE)
ON CONFLICT DO NOTHING;
INSERT INTO public.users (id, username, email, password_hash, role_id, store_id, is_active, reports_to, language, theme, must_change_password)
VALUES (4, 'cashier', 'cashier@retailpos.local', crypt('admin123', gen_salt('bf', 14)), 4, 1, TRUE, NULL, 'id', 'light', TRUE)
ON CONFLICT DO NOTHING;
INSERT INTO public.users (id, username, email, password_hash, role_id, store_id, is_active, reports_to, language, theme, must_change_password)
VALUES (5, 'inventory_staff', 'inventory_staff@retailpos.local', crypt('admin123', gen_salt('bf', 14)), 5, 1, TRUE, 3, 'id', 'light', TRUE)
ON CONFLICT DO NOTHING;
INSERT INTO public.users (id, username, email, password_hash, role_id, store_id, is_active, reports_to, language, theme, must_change_password)
VALUES (6, 'finance', 'finance@retailpos.local', crypt('admin123', gen_salt('bf', 14)), 6, 1, TRUE, NULL, 'id', 'light', TRUE)
ON CONFLICT DO NOTHING;

-- role_permissions (264 rows across 6 roles), resolved by role name and
-- permission code rather than by id, so the section stays correct
-- even where ids differ on an upgraded database.

-- cashier: 19 permissions
INSERT INTO public.role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM public.roles r
JOIN public.permissions p ON p.code = ANY (ARRAY['category.view', 'customer.view', 'customer_group.view', 'dashboard.view', 'pricing.view', 'product.view', 'receipt.print', 'sale.create', 'sale.detail', 'sale.lookup', 'sale.park', 'sale.view', 'shift.cash_movement', 'shift.create', 'shift.view', 'stock_opname.count', 'stock_opname.submit', 'stock_opname.view', 'storage_location.view'])
WHERE r.name = 'cashier'
ON CONFLICT DO NOTHING;

-- finance: 6 permissions
INSERT INTO public.role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM public.roles r
JOIN public.permissions p ON p.code = ANY (ARRAY['audit.view', 'consignment.pay', 'consignment.view', 'dashboard.view', 'report.view', 'sale.view'])
WHERE r.name = 'finance'
ON CONFLICT DO NOTHING;

-- inventory_staff: 17 permissions
INSERT INTO public.role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM public.roles r
JOIN public.permissions p ON p.code = ANY (ARRAY['inventory.adjust', 'stock_opname.assign', 'stock_opname.cancel', 'stock_opname.close', 'stock_opname.count', 'stock_opname.create', 'stock_opname.export', 'stock_opname.post', 'stock_opname.recount', 'stock_opname.report', 'stock_opname.submit', 'stock_opname.verify', 'stock_opname.view', 'storage_location.create', 'storage_location.delete', 'storage_location.update', 'storage_location.view'])
WHERE r.name = 'inventory_staff'
ON CONFLICT DO NOTHING;

-- manager: 79 permissions
INSERT INTO public.role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM public.roles r
JOIN public.permissions p ON p.code = ANY (ARRAY['app_settings.view', 'audit.export', 'audit.view', 'category.create', 'category.delete', 'category.export', 'category.import', 'category.update', 'category.view', 'consignment.create', 'consignment.pay', 'consignment.settle', 'consignment.update', 'consignment.view', 'customer.create', 'customer.delete', 'customer.export', 'customer.import', 'customer.update', 'customer.view', 'customer_group.create', 'customer_group.delete', 'customer_group.update', 'customer_group.view', 'dashboard.view', 'inventory.adjust', 'pricing.create', 'pricing.delete', 'pricing.update', 'pricing.view', 'product.cost.view', 'product.create', 'product.delete', 'product.export', 'product.history.view', 'product.import', 'product.update', 'product.view', 'purchase_order.cancel', 'purchase_order.confirm', 'purchase_order.create', 'purchase_order.receive', 'purchase_order.update', 'purchase_order.view', 'receipt.print', 'report.view', 'role.create', 'role.view', 'sale.create', 'sale.detail', 'sale.park', 'sale.view', 'shift.audit', 'shift.cash_movement', 'shift.create', 'shift.review', 'shift.view', 'stock_opname.assign', 'stock_opname.cancel', 'stock_opname.close', 'stock_opname.count', 'stock_opname.create', 'stock_opname.export', 'stock_opname.post', 'stock_opname.recount', 'stock_opname.report', 'stock_opname.submit', 'stock_opname.verify', 'stock_opname.view', 'storage_location.create', 'storage_location.delete', 'storage_location.update', 'storage_location.view', 'store.delete', 'store.update', 'store.view', 'user.create', 'user.update', 'user.view'])
WHERE r.name = 'manager'
ON CONFLICT DO NOTHING;

-- superadmin: 85 permissions
INSERT INTO public.role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM public.roles r
JOIN public.permissions p ON p.code = ANY (ARRAY['app_settings.update', 'app_settings.view', 'audit.export', 'audit.view', 'category.create', 'category.delete', 'category.export', 'category.import', 'category.update', 'category.view', 'consignment.create', 'consignment.pay', 'consignment.settle', 'consignment.update', 'consignment.view', 'customer.create', 'customer.delete', 'customer.export', 'customer.import', 'customer.update', 'customer.view', 'customer_group.create', 'customer_group.delete', 'customer_group.update', 'customer_group.view', 'dashboard.view', 'inventory.adjust', 'pricing.create', 'pricing.delete', 'pricing.update', 'pricing.view', 'product.cost.view', 'product.create', 'product.delete', 'product.export', 'product.history.view', 'product.import', 'product.update', 'product.view', 'purchase_order.cancel', 'purchase_order.confirm', 'purchase_order.create', 'purchase_order.delete', 'purchase_order.receive', 'purchase_order.update', 'purchase_order.view', 'receipt.print', 'report.view', 'role.create', 'role.delete', 'role.update', 'role.view', 'sale.create', 'sale.detail', 'sale.park', 'sale.view', 'shift.audit', 'shift.cash_movement', 'shift.create', 'shift.review', 'shift.view', 'stock_opname.assign', 'stock_opname.cancel', 'stock_opname.close', 'stock_opname.count', 'stock_opname.create', 'stock_opname.export', 'stock_opname.post', 'stock_opname.recount', 'stock_opname.report', 'stock_opname.submit', 'stock_opname.verify', 'stock_opname.view', 'storage_location.create', 'storage_location.delete', 'storage_location.update', 'storage_location.view', 'store.create', 'store.delete', 'store.update', 'store.view', 'user.create', 'user.delete', 'user.update', 'user.view'])
WHERE r.name = 'superadmin'
ON CONFLICT DO NOTHING;

-- supervisor: 58 permissions
INSERT INTO public.role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM public.roles r
JOIN public.permissions p ON p.code = ANY (ARRAY['category.create', 'category.delete', 'category.update', 'category.view', 'consignment.create', 'consignment.settle', 'consignment.update', 'consignment.view', 'customer.create', 'customer.delete', 'customer.export', 'customer.import', 'customer.update', 'customer.view', 'customer_group.create', 'customer_group.delete', 'customer_group.update', 'customer_group.view', 'dashboard.view', 'inventory.adjust', 'pricing.create', 'pricing.delete', 'pricing.update', 'pricing.view', 'product.cost.view', 'product.create', 'product.update', 'product.view', 'purchase_order.cancel', 'purchase_order.confirm', 'purchase_order.create', 'purchase_order.receive', 'purchase_order.update', 'purchase_order.view', 'receipt.print', 'report.view', 'sale.create', 'sale.detail', 'sale.park', 'sale.view', 'shift.audit', 'shift.cash_movement', 'shift.create', 'shift.review', 'shift.view', 'stock_opname.assign', 'stock_opname.cancel', 'stock_opname.close', 'stock_opname.count', 'stock_opname.create', 'stock_opname.export', 'stock_opname.post', 'stock_opname.recount', 'stock_opname.report', 'stock_opname.submit', 'stock_opname.verify', 'stock_opname.view', 'storage_location.view'])
WHERE r.name = 'supervisor'
ON CONFLICT DO NOTHING;

-- 044_store_first_and_finance_role.sql step 7 backfills store_id
-- for store-scoped roles. superadmin is deliberately excluded (it is
-- an HQ account and stays store-less), so the role list below matches
-- 044 exactly. The inserts above already carry the right values; this
-- only matters on a database seeded before 044.

UPDATE public.users u
SET store_id = (SELECT s.id FROM public.stores s ORDER BY s.id LIMIT 1)
WHERE u.store_id IS NULL
  AND u.role_id IN (SELECT r.id FROM public.roles r
                    WHERE r.name IN ('supervisor', 'manager', 'cashier',
                                      'finance', 'inventory_staff'))
  AND EXISTS (SELECT 1 FROM public.stores);

-- ============================================================================
-- Reference data self-check
-- ============================================================================
-- Fails loudly if the seed data above did not land, so a partial apply can
-- never be mistaken for a working install. Counts that may legitimately be
-- higher on a database that was already in use are checked with < rather
-- than =.
-- ============================================================================
DO $$
DECLARE
    n       bigint;
    missing text := '';
BEGIN
    SELECT count(*) INTO n FROM roles WHERE name IN
        ('superadmin', 'manager', 'supervisor', 'cashier',
         'inventory_staff', 'finance');
    IF n <> 6 THEN
        missing := missing || format(' roles=%s/6', n);
    END IF;

    SELECT count(*) INTO n FROM users WHERE username IN
        ('superadmin', 'manager', 'supervisor', 'cashier',
         'inventory_staff', 'finance');
    IF n <> 6 THEN
        missing := missing || format(' users=%s/6', n);
    END IF;

    -- The published seed credential must verify against every system account.
    SELECT count(*) INTO n FROM users
    WHERE username IN ('superadmin', 'manager', 'supervisor', 'cashier',
                       'inventory_staff', 'finance')
      AND password_hash = crypt('admin123', password_hash);
    IF n <> 6 THEN
        missing := missing || format(' seed credential verified %s/6', n);
    END IF;

    -- 052_first_install_hardening.sql forces rotation for exactly these six.
    SELECT count(*) INTO n FROM users
    WHERE username IN ('superadmin', 'manager', 'supervisor', 'cashier',
                       'inventory_staff', 'finance')
      AND must_change_password;
    IF n <> 6 THEN
        missing := missing || format(' must_change_password %s/6', n);
    END IF;

    SELECT count(*) INTO n FROM permissions;
    IF n < 86 THEN
        missing := missing || format(' permissions=%s/86', n);
    END IF;

    SELECT count(*) INTO n FROM role_permissions;
    IF n < 264 THEN
        missing := missing || format(' role_permissions=%s/264', n);
    END IF;

    SELECT count(*) INTO n FROM payment_methods;
    IF n < 5 THEN
        missing := missing || format(' payment_methods=%s/5', n);
    END IF;

    SELECT count(*) INTO n FROM customer_groups;
    IF n < 3 THEN
        missing := missing || format(' customer_groups=%s/3', n);
    END IF;

    SELECT count(*) INTO n FROM app_settings;
    IF n < 7 THEN
        missing := missing || format(' app_settings=%s/7', n);
    END IF;

    -- id = 1 is the default store; tests and the seeder hard-code StoreID: 1.
    SELECT count(*) INTO n FROM stores WHERE id = 1;
    IF n <> 1 THEN
        missing := missing || ' stores(id=1)';
    END IF;

    -- id = 1 is the walk-in customer checkout falls back to.
    SELECT count(*) INTO n FROM customers WHERE id = 1 AND is_walk_in;
    IF n <> 1 THEN
        missing := missing || ' customers(id=1, walk-in)';
    END IF;

    IF missing <> '' THEN
        RAISE EXCEPTION '000_baseline.sql: incomplete reference data:%', missing;
    END IF;
END $$;

-- ============================================================================
-- Ledger reconciliation
-- ============================================================================
-- The three non-test runners replay every file in database/migrations/*.sql on
-- every run and then record it; internal/shared/testdb.go instead skips files
-- already present in schema_migrations and requires
-- len(schema_migrations) == len(files).
--
-- After the squash there is exactly one migration file, so the rows recorded
-- for the 32 migrations this file replaces must be dropped or that count never
-- converges. The rows are listed explicitly rather than deleted wholesale so
-- that migrations added after this baseline keep their own entries.
--
-- This block stays last: if anything above fails, the old rows survive and
-- testdb.go replays this file instead of believing the database is current.
-- ============================================================================
DELETE FROM schema_migrations
WHERE filename IN (
    '000_squash.sql',
    '001_consignment.sql',
    '002_settlement_items_product_id.sql',
    '003_settlement_updated_at.sql',
    '004_supplier_code_sequence.sql',
    '005_app_settings.sql',
    '006_user_preferences.sql',
    '007_sale_lookup.sql',
    '031_revoke_sale_lookup_manager.sql',
    '032_sale_detail_and_receipt_print.sql',
    '033_audit_log_store_and_immutability.sql',
    '033b_cash_change.sql',
    '034_audit_immutable_bypass.sql',
    '035_audit_correlation_id.sql',
    '036_audit_export_permission.sql',
    '037_audit_immutable_fk_bypass.sql',
    '038_grant_audit_view_to_admin.sql',
    '039_business_permission_audit.sql',
    '040_shift_cash_movements.sql',
    '041_shift_settings.sql',
    '042_consignment_receipt_edit.sql',
    '043_product_ownership_type.sql',
    '044_store_first_and_finance_role.sql',
    '045_rename_usernames.sql',
    '046_manager_consignment_pay.sql',
    '047_finance_consignment_view.sql',
    '048_add_termination_return_reason.sql',
    '049_revoke_store_view_finance_supervisor.sql',
    '050_store_onboarding.sql',
    '051_pagination_indexes.sql',
    '052_first_install_hardening.sql',
    '053_restore_core_foreign_keys.sql'
);

INSERT INTO schema_migrations (filename)
VALUES ('000_baseline.sql')
ON CONFLICT (filename) DO NOTHING;

COMMIT;
