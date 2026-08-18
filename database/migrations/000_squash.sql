-- Migration: 000_squash.sql
-- Description: Version 1 baseline — idempotent final schema covering the full
-- migration history (previously 000_squash + 001..030). Generated from
-- pg_dump schema-only of a fully-migrated reference database.
-- Usage: CREATE TABLE IF NOT EXISTS + CREATE INDEX IF NOT EXISTS +
--        INSERT ... ON CONFLICT DO NOTHING for safe re-runs.

BEGIN;

-- ============================================================
-- Extensions
-- ============================================================
CREATE EXTENSION IF NOT EXISTS pg_trgm WITH SCHEMA public;
CREATE EXTENSION IF NOT EXISTS pgcrypto WITH SCHEMA public;

-- ============================================================
-- Functions
-- ============================================================
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

-- ============================================================
-- Standalone sequences (not owned by a table column)
-- ============================================================
CREATE SEQUENCE IF NOT EXISTS do_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

CREATE SEQUENCE IF NOT EXISTS gr_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

CREATE SEQUENCE IF NOT EXISTS ia_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

CREATE SEQUENCE IF NOT EXISTS invoice_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

CREATE SEQUENCE IF NOT EXISTS po_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

CREATE SEQUENCE IF NOT EXISTS sku_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

CREATE SEQUENCE IF NOT EXISTS so_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

-- ============================================================
-- Tables
-- ============================================================
CREATE TABLE IF NOT EXISTS brands (
    id SERIAL,
    name character varying(100) NOT NULL,
    description text,
    is_active boolean DEFAULT true,
    created_at timestamp with time zone DEFAULT now(),
    updated_at timestamp with time zone DEFAULT now(),
    CONSTRAINT brands_name_key UNIQUE (name),
    CONSTRAINT brands_pkey PRIMARY KEY (id)
);

CREATE TABLE IF NOT EXISTS categories (
    id SERIAL,
    name character varying(100) NOT NULL,
    description text,
    parent_id integer,
    slug character varying(120),
    is_active boolean DEFAULT true,
    created_at timestamp with time zone DEFAULT now(),
    updated_at timestamp with time zone DEFAULT now(),
    CONSTRAINT categories_name_key UNIQUE (name),
    CONSTRAINT categories_pkey PRIMARY KEY (id),
    CONSTRAINT categories_slug_key UNIQUE (slug),
    CONSTRAINT categories_parent_id_fkey FOREIGN KEY (parent_id) REFERENCES categories(id) ON DELETE SET NULL
);

CREATE TABLE IF NOT EXISTS customer_groups (
    id SERIAL,
    name character varying(100) NOT NULL,
    description text,
    is_active boolean DEFAULT true NOT NULL,
    color character varying(7),
    created_at timestamp with time zone DEFAULT now(),
    updated_at timestamp with time zone DEFAULT now(),
    CONSTRAINT customer_groups_name_key UNIQUE (name),
    CONSTRAINT customer_groups_pkey PRIMARY KEY (id)
);

CREATE TABLE IF NOT EXISTS dead_letter_events (
    id BIGSERIAL,
    event_type text NOT NULL,
    payload jsonb,
    error text NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT dead_letter_events_pkey PRIMARY KEY (id)
);

CREATE TABLE IF NOT EXISTS payment_methods (
    id SERIAL,
    code character varying(30) NOT NULL,
    name character varying(100) NOT NULL,
    is_active boolean DEFAULT true,
    requires_reference boolean DEFAULT false,
    sort_order integer DEFAULT 0 NOT NULL,
    created_at timestamp with time zone DEFAULT now(),
    updated_at timestamp with time zone DEFAULT now(),
    CONSTRAINT payment_methods_code_key UNIQUE (code),
    CONSTRAINT payment_methods_pkey PRIMARY KEY (id)
);

CREATE TABLE IF NOT EXISTS permissions (
    id SERIAL,
    code character varying(50) NOT NULL,
    name character varying(100) NOT NULL,
    description text,
    created_at timestamp with time zone DEFAULT now(),
    CONSTRAINT permissions_code_key UNIQUE (code),
    CONSTRAINT permissions_pkey PRIMARY KEY (id)
);

CREATE TABLE IF NOT EXISTS roles (
    id SERIAL,
    name character varying(50) NOT NULL,
    description text,
    is_system boolean DEFAULT false,
    created_at timestamp with time zone DEFAULT now(),
    CONSTRAINT roles_name_key UNIQUE (name),
    CONSTRAINT roles_pkey PRIMARY KEY (id)
);

CREATE TABLE IF NOT EXISTS stores (
    id SERIAL,
    name character varying(100) NOT NULL,
    address text,
    phone character varying(20),
    is_active boolean DEFAULT true,
    created_at timestamp with time zone DEFAULT now(),
    CONSTRAINT stores_pkey PRIMARY KEY (id)
);

CREATE TABLE IF NOT EXISTS tax_classes (
    id SERIAL,
    name character varying(100) NOT NULL,
    rate_percent numeric(5,2) NOT NULL,
    description text,
    is_active boolean DEFAULT true,
    created_at timestamp with time zone DEFAULT now(),
    CONSTRAINT tax_classes_name_key UNIQUE (name),
    CONSTRAINT tax_classes_pkey PRIMARY KEY (id)
);

CREATE TABLE IF NOT EXISTS units_of_measure (
    id SERIAL,
    code character varying(10) NOT NULL,
    name character varying(100) NOT NULL,
    description text,
    is_active boolean DEFAULT true,
    created_at timestamp with time zone DEFAULT now(),
    CONSTRAINT units_of_measure_code_key UNIQUE (code),
    CONSTRAINT units_of_measure_pkey PRIMARY KEY (id)
);

CREATE TABLE IF NOT EXISTS users (
    id SERIAL,
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
    CONSTRAINT users_email_key UNIQUE (email),
    CONSTRAINT users_pkey PRIMARY KEY (id),
    CONSTRAINT users_username_key UNIQUE (username),
    CONSTRAINT users_reports_to_fkey FOREIGN KEY (reports_to) REFERENCES users(id) ON DELETE SET NULL
);

CREATE TABLE IF NOT EXISTS audit_logs (
    id SERIAL,
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
    CONSTRAINT audit_logs_pkey PRIMARY KEY (id),
    CONSTRAINT audit_logs_user_id_fkey FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE SET NULL
);

CREATE TABLE IF NOT EXISTS customers (
    id SERIAL,
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
    updated_at timestamp with time zone DEFAULT now(),
    CONSTRAINT customers_phone_key UNIQUE (phone),
    CONSTRAINT customers_pkey PRIMARY KEY (id),
    CONSTRAINT customers_customer_group_id_fkey FOREIGN KEY (customer_group_id) REFERENCES customer_groups(id) ON DELETE SET NULL
);

CREATE TABLE IF NOT EXISTS import_jobs (
    id BIGSERIAL,
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
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT import_jobs_pkey PRIMARY KEY (id),
    CONSTRAINT import_jobs_store_id_fkey FOREIGN KEY (store_id) REFERENCES stores(id),
    CONSTRAINT import_jobs_user_id_fkey FOREIGN KEY (user_id) REFERENCES users(id)
);

CREATE TABLE IF NOT EXISTS products (
    id SERIAL,
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
    CONSTRAINT chk_product_status CHECK (((status)::text = ANY ((ARRAY['draft'::character varying, 'active'::character varying, 'inactive'::character varying, 'discontinued'::character varying, 'archived'::character varying])::text[]))),
    CONSTRAINT products_cost_check CHECK ((cost >= 0)),
    CONSTRAINT products_price_check CHECK ((price >= 0)),
    CONSTRAINT products_pkey PRIMARY KEY (id),
    CONSTRAINT products_sku_key UNIQUE (sku),
    CONSTRAINT products_brand_id_fkey FOREIGN KEY (brand_id) REFERENCES brands(id) ON DELETE SET NULL,
    CONSTRAINT products_category_id_fkey FOREIGN KEY (category_id) REFERENCES categories(id) ON DELETE RESTRICT,
    CONSTRAINT products_store_id_fkey FOREIGN KEY (store_id) REFERENCES stores(id) ON DELETE SET NULL,
    CONSTRAINT products_tax_class_id_fkey FOREIGN KEY (tax_class_id) REFERENCES tax_classes(id) ON DELETE SET NULL,
    CONSTRAINT products_unit_of_measure_id_fkey FOREIGN KEY (unit_of_measure_id) REFERENCES units_of_measure(id) ON DELETE SET NULL
);

CREATE TABLE IF NOT EXISTS refresh_tokens (
    id SERIAL,
    user_id integer NOT NULL,
    token_hash text NOT NULL,
    expires_at timestamp with time zone NOT NULL,
    created_at timestamp with time zone DEFAULT now(),
    CONSTRAINT refresh_tokens_pkey PRIMARY KEY (id),
    CONSTRAINT refresh_tokens_token_hash_key UNIQUE (token_hash),
    CONSTRAINT refresh_tokens_user_id_fkey FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS role_permissions (
    role_id integer NOT NULL,
    permission_id integer NOT NULL,
    CONSTRAINT role_permissions_pkey PRIMARY KEY (role_id, permission_id),
    CONSTRAINT role_permissions_permission_id_fkey FOREIGN KEY (permission_id) REFERENCES permissions(id) ON DELETE CASCADE,
    CONSTRAINT role_permissions_role_id_fkey FOREIGN KEY (role_id) REFERENCES roles(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS shifts (
    id SERIAL,
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
    CONSTRAINT chk_shift_status CHECK (((status)::text = ANY ((ARRAY['open'::character varying, 'closed'::character varying])::text[]))),
    CONSTRAINT shifts_pkey PRIMARY KEY (id),
    CONSTRAINT shifts_reviewed_by_fkey FOREIGN KEY (reviewed_by) REFERENCES users(id),
    CONSTRAINT shifts_store_id_fkey FOREIGN KEY (store_id) REFERENCES stores(id) ON DELETE SET NULL,
    CONSTRAINT shifts_user_id_fkey FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE RESTRICT
);

CREATE TABLE IF NOT EXISTS suppliers (
    id SERIAL,
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
    CONSTRAINT suppliers_code_key UNIQUE (code),
    CONSTRAINT suppliers_pkey PRIMARY KEY (id),
    CONSTRAINT suppliers_store_id_fkey FOREIGN KEY (store_id) REFERENCES stores(id)
);

CREATE TABLE IF NOT EXISTS warehouses (
    id SERIAL,
    name character varying(100) NOT NULL,
    code character varying(20) NOT NULL,
    address text,
    store_id integer,
    is_active boolean DEFAULT true,
    created_at timestamp with time zone DEFAULT now(),
    CONSTRAINT warehouses_code_key UNIQUE (code),
    CONSTRAINT warehouses_pkey PRIMARY KEY (id),
    CONSTRAINT warehouses_store_id_fkey FOREIGN KEY (store_id) REFERENCES stores(id) ON DELETE SET NULL
);

CREATE TABLE IF NOT EXISTS cart_sessions (
    id SERIAL,
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
    CONSTRAINT cart_sessions_status_check CHECK (((status)::text = ANY ((ARRAY['open'::character varying, 'held'::character varying, 'checked_out'::character varying, 'cancelled'::character varying, 'expired'::character varying])::text[]))),
    CONSTRAINT cart_sessions_pkey PRIMARY KEY (id),
    CONSTRAINT cart_sessions_cashier_id_fkey FOREIGN KEY (cashier_id) REFERENCES users(id) ON DELETE RESTRICT,
    CONSTRAINT cart_sessions_customer_id_fkey FOREIGN KEY (customer_id) REFERENCES customers(id),
    CONSTRAINT cart_sessions_shift_id_fkey FOREIGN KEY (shift_id) REFERENCES shifts(id),
    CONSTRAINT cart_sessions_store_id_fkey FOREIGN KEY (store_id) REFERENCES stores(id) ON DELETE SET NULL
);

CREATE TABLE IF NOT EXISTS import_errors (
    id BIGSERIAL,
    import_job_id bigint NOT NULL,
    row_number integer NOT NULL,
    field character varying(100),
    value text,
    reason text NOT NULL,
    suggestion text,
    stage character varying(30) NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT import_errors_pkey PRIMARY KEY (id),
    CONSTRAINT import_errors_import_job_id_fkey FOREIGN KEY (import_job_id) REFERENCES import_jobs(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS import_rows (
    id BIGSERIAL,
    import_job_id bigint NOT NULL,
    row_number integer NOT NULL,
    status character varying(20) NOT NULL,
    entity_id integer,
    old_values jsonb,
    new_values jsonb,
    changed_fields text[],
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT import_rows_pkey PRIMARY KEY (id),
    CONSTRAINT import_rows_import_job_id_fkey FOREIGN KEY (import_job_id) REFERENCES import_jobs(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS import_snapshots (
    id BIGSERIAL,
    import_job_id bigint NOT NULL,
    rows_data jsonb NOT NULL,
    schema_snapshot jsonb NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT import_snapshots_pkey PRIMARY KEY (id),
    CONSTRAINT import_snapshots_import_job_id_fkey FOREIGN KEY (import_job_id) REFERENCES import_jobs(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS inventory_movements (
    id SERIAL,
    product_id integer NOT NULL,
    quantity_change integer NOT NULL,
    type character varying(50) NOT NULL,
    reference_id integer,
    reference_table character varying(50),
    user_id integer,
    notes text,
    created_at timestamp with time zone DEFAULT now(),
    CONSTRAINT inventory_movements_pkey PRIMARY KEY (id),
    CONSTRAINT inventory_movements_product_id_fkey FOREIGN KEY (product_id) REFERENCES products(id) ON DELETE CASCADE,
    CONSTRAINT inventory_movements_user_id_fkey FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE SET NULL
);

CREATE TABLE IF NOT EXISTS pricing_rules (
    id SERIAL,
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
    CONSTRAINT chk_pricing_status CHECK (((status)::text = ANY ((ARRAY['draft'::character varying, 'pending'::character varying, 'approved'::character varying, 'rejected'::character varying])::text[]))),
    CONSTRAINT chk_pricing_target CHECK (((product_id IS NOT NULL) OR (category_id IS NOT NULL) OR (brand_id IS NOT NULL))),
    CONSTRAINT chk_pricing_type CHECK (((pricing_type)::text = ANY ((ARRAY['special_price'::character varying, 'promotion'::character varying])::text[]))),
    CONSTRAINT pricing_rules_minimum_quantity_check CHECK ((minimum_quantity >= 1)),
    CONSTRAINT pricing_rules_name_unique UNIQUE (name),
    CONSTRAINT pricing_rules_pkey PRIMARY KEY (id),
    CONSTRAINT pricing_rules_brand_id_fkey FOREIGN KEY (brand_id) REFERENCES brands(id) ON DELETE CASCADE,
    CONSTRAINT pricing_rules_category_id_fkey FOREIGN KEY (category_id) REFERENCES categories(id) ON DELETE CASCADE,
    CONSTRAINT pricing_rules_customer_group_id_fkey FOREIGN KEY (customer_group_id) REFERENCES customer_groups(id) ON DELETE SET NULL,
    CONSTRAINT pricing_rules_product_id_fkey FOREIGN KEY (product_id) REFERENCES products(id) ON DELETE CASCADE,
    CONSTRAINT pricing_rules_store_id_fkey FOREIGN KEY (store_id) REFERENCES stores(id) ON DELETE SET NULL
);

CREATE TABLE IF NOT EXISTS product_suppliers (
    id SERIAL,
    product_id integer NOT NULL,
    supplier_id integer NOT NULL,
    supplier_sku character varying(50),
    unit_cost integer DEFAULT 0,
    lead_time_days integer DEFAULT 0,
    is_preferred boolean DEFAULT false,
    created_at timestamp with time zone DEFAULT now(),
    updated_at timestamp with time zone DEFAULT now(),
    CONSTRAINT product_suppliers_unit_cost_check CHECK ((unit_cost >= 0)),
    CONSTRAINT product_suppliers_pkey PRIMARY KEY (id),
    CONSTRAINT product_suppliers_product_id_supplier_id_key UNIQUE (product_id, supplier_id),
    CONSTRAINT product_suppliers_product_id_fkey FOREIGN KEY (product_id) REFERENCES products(id) ON DELETE CASCADE,
    CONSTRAINT product_suppliers_supplier_id_fkey FOREIGN KEY (supplier_id) REFERENCES suppliers(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS purchase_orders (
    id SERIAL,
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
    approved_at timestamp with time zone,
    CONSTRAINT purchase_orders_pkey PRIMARY KEY (id),
    CONSTRAINT purchase_orders_po_number_key UNIQUE (po_number),
    CONSTRAINT purchase_orders_supplier_id_fkey FOREIGN KEY (supplier_id) REFERENCES suppliers(id)
);

CREATE TABLE IF NOT EXISTS sales (
    id SERIAL,
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
    CONSTRAINT sales_invoice_number_key UNIQUE (invoice_number),
    CONSTRAINT sales_pkey PRIMARY KEY (id),
    CONSTRAINT sales_cashier_id_fkey FOREIGN KEY (cashier_id) REFERENCES users(id) ON DELETE RESTRICT,
    CONSTRAINT sales_customer_id_fkey FOREIGN KEY (customer_id) REFERENCES customers(id),
    CONSTRAINT sales_shift_id_fkey FOREIGN KEY (shift_id) REFERENCES shifts(id),
    CONSTRAINT sales_store_id_fkey FOREIGN KEY (store_id) REFERENCES stores(id) ON DELETE SET NULL
);

CREATE TABLE IF NOT EXISTS storage_locations (
    id SERIAL,
    code character varying(50) NOT NULL,
    name character varying(100) NOT NULL,
    warehouse_id integer,
    store_id integer,
    notes text,
    is_active boolean DEFAULT true NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT chk_storage_locations_scope CHECK (((warehouse_id IS NOT NULL) OR (store_id IS NOT NULL))),
    CONSTRAINT storage_locations_pkey PRIMARY KEY (id),
    CONSTRAINT uq_storage_locations_code UNIQUE (code),
    CONSTRAINT storage_locations_store_id_fkey FOREIGN KEY (store_id) REFERENCES stores(id) ON DELETE SET NULL,
    CONSTRAINT storage_locations_warehouse_id_fkey FOREIGN KEY (warehouse_id) REFERENCES warehouses(id) ON DELETE SET NULL
);

CREATE TABLE IF NOT EXISTS cart_items (
    id SERIAL,
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
    CONSTRAINT chk_cart_item_subtotal CHECK ((subtotal = (quantity * unit_price))),
    CONSTRAINT cart_items_pkey PRIMARY KEY (id),
    CONSTRAINT cart_items_cart_session_id_fkey FOREIGN KEY (cart_session_id) REFERENCES cart_sessions(id) ON DELETE CASCADE,
    CONSTRAINT cart_items_pricing_rule_id_fkey FOREIGN KEY (pricing_rule_id) REFERENCES pricing_rules(id) ON DELETE SET NULL,
    CONSTRAINT cart_items_product_id_fkey FOREIGN KEY (product_id) REFERENCES products(id) ON DELETE RESTRICT,
    CONSTRAINT cart_items_tax_class_id_fkey FOREIGN KEY (tax_class_id) REFERENCES tax_classes(id) ON DELETE SET NULL
);

CREATE TABLE IF NOT EXISTS goods_receipts (
    id SERIAL,
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
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT goods_receipts_gr_number_key UNIQUE (gr_number),
    CONSTRAINT goods_receipts_pkey PRIMARY KEY (id),
    CONSTRAINT goods_receipts_purchase_order_id_fkey FOREIGN KEY (purchase_order_id) REFERENCES purchase_orders(id)
);

CREATE TABLE IF NOT EXISTS product_stock (
    id SERIAL,
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
    CONSTRAINT product_stock_reorder_quantity_check CHECK ((reorder_quantity >= 0)),
    CONSTRAINT product_stock_pkey PRIMARY KEY (id),
    CONSTRAINT uq_product_stock UNIQUE NULLS NOT DISTINCT (product_id, warehouse_id, store_id, location_id),
    CONSTRAINT product_stock_location_id_fkey FOREIGN KEY (location_id) REFERENCES storage_locations(id) ON DELETE SET NULL,
    CONSTRAINT product_stock_product_id_fkey FOREIGN KEY (product_id) REFERENCES products(id) ON DELETE CASCADE,
    CONSTRAINT product_stock_store_id_fkey FOREIGN KEY (store_id) REFERENCES stores(id) ON DELETE SET NULL,
    CONSTRAINT product_stock_warehouse_id_fkey FOREIGN KEY (warehouse_id) REFERENCES warehouses(id) ON DELETE SET NULL
);

CREATE TABLE IF NOT EXISTS purchase_order_items (
    id SERIAL,
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
    CONSTRAINT purchase_order_items_unit_cost_check CHECK ((unit_cost >= 0)),
    CONSTRAINT purchase_order_items_pkey PRIMARY KEY (id),
    CONSTRAINT purchase_order_items_purchase_order_id_product_id_key UNIQUE (purchase_order_id, product_id),
    CONSTRAINT purchase_order_items_product_id_fkey FOREIGN KEY (product_id) REFERENCES products(id),
    CONSTRAINT purchase_order_items_purchase_order_id_fkey FOREIGN KEY (purchase_order_id) REFERENCES purchase_orders(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS sale_items (
    id SERIAL,
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
    CONSTRAINT sale_items_unit_price_check CHECK ((unit_price >= 0)),
    CONSTRAINT sale_items_pkey PRIMARY KEY (id),
    CONSTRAINT sale_items_pricing_rule_id_fkey FOREIGN KEY (pricing_rule_id) REFERENCES pricing_rules(id) ON DELETE SET NULL,
    CONSTRAINT sale_items_product_id_fkey FOREIGN KEY (product_id) REFERENCES products(id) ON DELETE RESTRICT,
    CONSTRAINT sale_items_sale_id_fkey FOREIGN KEY (sale_id) REFERENCES sales(id) ON DELETE CASCADE,
    CONSTRAINT sale_items_tax_class_id_fkey FOREIGN KEY (tax_class_id) REFERENCES tax_classes(id) ON DELETE SET NULL
);

CREATE TABLE IF NOT EXISTS sale_payments (
    id SERIAL,
    sale_id integer NOT NULL,
    payment_method_id integer NOT NULL,
    payment_method_code character varying(30) NOT NULL,
    amount integer NOT NULL,
    reference_number character varying(100),
    created_at timestamp with time zone DEFAULT now(),
    CONSTRAINT sale_payments_amount_check CHECK ((amount > 0)),
    CONSTRAINT sale_payments_pkey PRIMARY KEY (id),
    CONSTRAINT sale_payments_payment_method_id_fkey FOREIGN KEY (payment_method_id) REFERENCES payment_methods(id) ON DELETE RESTRICT,
    CONSTRAINT sale_payments_sale_id_fkey FOREIGN KEY (sale_id) REFERENCES sales(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS stock_opnames (
    id SERIAL,
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
    CONSTRAINT chk_stock_opname_scope_type CHECK (((scope_type)::text = ANY ((ARRAY['store'::character varying, 'warehouse'::character varying, 'category'::character varying, 'brand'::character varying, 'supplier'::character varying, 'product'::character varying, 'manual'::character varying, 'location'::character varying])::text[]))),
    CONSTRAINT chk_stock_opname_status CHECK (((status)::text = ANY ((ARRAY['draft'::character varying, 'open'::character varying, 'counting'::character varying, 'verification'::character varying, 'needs_recount'::character varying, 'approved'::character varying, 'posted'::character varying, 'closed'::character varying, 'cancelled'::character varying])::text[]))),
    CONSTRAINT stock_opnames_pkey PRIMARY KEY (id),
    CONSTRAINT stock_opnames_session_number_key UNIQUE (session_number),
    CONSTRAINT stock_opnames_closed_by_fkey FOREIGN KEY (closed_by) REFERENCES users(id),
    CONSTRAINT stock_opnames_location_id_fkey FOREIGN KEY (location_id) REFERENCES storage_locations(id) ON DELETE SET NULL,
    CONSTRAINT stock_opnames_opened_by_fkey FOREIGN KEY (opened_by) REFERENCES users(id),
    CONSTRAINT stock_opnames_posted_by_fkey FOREIGN KEY (posted_by) REFERENCES users(id),
    CONSTRAINT stock_opnames_store_id_fkey FOREIGN KEY (store_id) REFERENCES stores(id) ON DELETE SET NULL,
    CONSTRAINT stock_opnames_verified_by_fkey FOREIGN KEY (verified_by) REFERENCES users(id)
);

CREATE TABLE IF NOT EXISTS goods_receipt_items (
    id SERIAL,
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
    CONSTRAINT goods_receipt_items_qty_good_check CHECK ((qty_good >= 0)),
    CONSTRAINT goods_receipt_items_pkey PRIMARY KEY (id),
    CONSTRAINT goods_receipt_items_goods_receipt_id_fkey FOREIGN KEY (goods_receipt_id) REFERENCES goods_receipts(id) ON DELETE CASCADE,
    CONSTRAINT goods_receipt_items_product_id_fkey FOREIGN KEY (product_id) REFERENCES products(id),
    CONSTRAINT goods_receipt_items_purchase_order_item_id_fkey FOREIGN KEY (purchase_order_item_id) REFERENCES purchase_order_items(id)
);

CREATE TABLE IF NOT EXISTS inventory_adjustments (
    id BIGSERIAL,
    adjustment_number character varying(30) NOT NULL,
    session_id integer NOT NULL,
    status character varying(20) DEFAULT 'posted'::character varying NOT NULL,
    notes text,
    created_by integer,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT inventory_adjustments_adjustment_number_key UNIQUE (adjustment_number),
    CONSTRAINT inventory_adjustments_pkey PRIMARY KEY (id),
    CONSTRAINT inventory_adjustments_created_by_fkey FOREIGN KEY (created_by) REFERENCES users(id),
    CONSTRAINT inventory_adjustments_session_id_fkey FOREIGN KEY (session_id) REFERENCES stock_opnames(id) ON DELETE RESTRICT
);

CREATE TABLE IF NOT EXISTS stock_opname_assignments (
    id SERIAL,
    stock_opname_id integer NOT NULL,
    user_id integer NOT NULL,
    role character varying(20) NOT NULL,
    assigned_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT chk_stock_opname_assignment_role CHECK (((role)::text = ANY ((ARRAY['counter'::character varying, 'supervisor'::character varying])::text[]))),
    CONSTRAINT stock_opname_assignments_pkey PRIMARY KEY (id),
    CONSTRAINT stock_opname_assignments_stock_opname_id_fkey FOREIGN KEY (stock_opname_id) REFERENCES stock_opnames(id) ON DELETE CASCADE,
    CONSTRAINT stock_opname_assignments_user_id_fkey FOREIGN KEY (user_id) REFERENCES users(id)
);

CREATE TABLE IF NOT EXISTS stock_opname_items (
    id SERIAL,
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
    CONSTRAINT chk_stock_opname_item_status CHECK (((status)::text = ANY ((ARRAY['pending'::character varying, 'counted'::character varying])::text[]))),
    CONSTRAINT stock_opname_items_physical_qty_check CHECK ((physical_qty >= (0)::numeric)),
    CONSTRAINT stock_opname_items_pkey PRIMARY KEY (id),
    CONSTRAINT stock_opname_items_product_id_fkey FOREIGN KEY (product_id) REFERENCES products(id),
    CONSTRAINT stock_opname_items_stock_opname_id_fkey FOREIGN KEY (stock_opname_id) REFERENCES stock_opnames(id) ON DELETE CASCADE,
    CONSTRAINT stock_opname_items_store_id_fkey FOREIGN KEY (store_id) REFERENCES stores(id),
    CONSTRAINT stock_opname_items_warehouse_id_fkey FOREIGN KEY (warehouse_id) REFERENCES warehouses(id)
);

CREATE TABLE IF NOT EXISTS stock_opname_recount_requests (
    id BIGSERIAL,
    stock_opname_id integer NOT NULL,
    requested_by integer NOT NULL,
    reason text,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT stock_opname_recount_requests_pkey PRIMARY KEY (id),
    CONSTRAINT stock_opname_recount_requests_requested_by_fkey FOREIGN KEY (requested_by) REFERENCES users(id),
    CONSTRAINT stock_opname_recount_requests_stock_opname_id_fkey FOREIGN KEY (stock_opname_id) REFERENCES stock_opnames(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS stock_opname_session_scopes (
    id BIGSERIAL,
    stock_opname_id integer NOT NULL,
    scope_type character varying(30) NOT NULL,
    scope_id bigint,
    scope_name character varying(255) DEFAULT ''::character varying,
    scope_data jsonb,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT chk_so_scope_type CHECK (((scope_type)::text = ANY ((ARRAY['store'::character varying, 'warehouse'::character varying, 'category'::character varying, 'brand'::character varying, 'supplier'::character varying, 'product'::character varying, 'manual'::character varying, 'location'::character varying])::text[]))),
    CONSTRAINT stock_opname_session_scopes_pkey PRIMARY KEY (id),
    CONSTRAINT stock_opname_session_scopes_stock_opname_id_fkey FOREIGN KEY (stock_opname_id) REFERENCES stock_opnames(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS inventory_adjustment_items (
    id BIGSERIAL,
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
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT inventory_adjustment_items_pkey PRIMARY KEY (id),
    CONSTRAINT inventory_adjustment_items_adjustment_id_fkey FOREIGN KEY (adjustment_id) REFERENCES inventory_adjustments(id) ON DELETE CASCADE,
    CONSTRAINT inventory_adjustment_items_product_id_fkey FOREIGN KEY (product_id) REFERENCES products(id),
    CONSTRAINT inventory_adjustment_items_store_id_fkey FOREIGN KEY (store_id) REFERENCES stores(id),
    CONSTRAINT inventory_adjustment_items_warehouse_id_fkey FOREIGN KEY (warehouse_id) REFERENCES warehouses(id)
);

CREATE TABLE IF NOT EXISTS stock_opname_counts (
    id SERIAL,
    stock_opname_item_id integer NOT NULL,
    count_sequence integer NOT NULL,
    physical_qty numeric(18,4) NOT NULL,
    counted_by integer NOT NULL,
    counted_at timestamp with time zone DEFAULT now() NOT NULL,
    remarks text,
    CONSTRAINT stock_opname_counts_count_sequence_check CHECK ((count_sequence >= 1)),
    CONSTRAINT stock_opname_counts_physical_qty_check CHECK ((physical_qty >= (0)::numeric)),
    CONSTRAINT stock_opname_counts_pkey PRIMARY KEY (id),
    CONSTRAINT stock_opname_counts_counted_by_fkey FOREIGN KEY (counted_by) REFERENCES users(id),
    CONSTRAINT stock_opname_counts_stock_opname_item_id_fkey FOREIGN KEY (stock_opname_item_id) REFERENCES stock_opname_items(id) ON DELETE CASCADE
);

-- ============================================================
-- Indexes
-- ============================================================
CREATE INDEX IF NOT EXISTS idx_audit_logs_action_ip_created ON audit_logs USING btree (action, ip_address, created_at);

CREATE INDEX IF NOT EXISTS idx_audit_logs_created ON audit_logs USING btree (created_at);

CREATE INDEX IF NOT EXISTS idx_audit_logs_user ON audit_logs USING btree (user_id);

CREATE INDEX IF NOT EXISTS idx_cart_items_session ON cart_items USING btree (cart_session_id);

CREATE INDEX IF NOT EXISTS idx_cart_sessions_cashier_status ON cart_sessions USING btree (cashier_id, status);

CREATE INDEX IF NOT EXISTS idx_cart_sessions_shift ON cart_sessions USING btree (shift_id);

CREATE INDEX IF NOT EXISTS idx_categories_slug ON categories USING btree (slug);

CREATE INDEX IF NOT EXISTS idx_customer_groups_active ON customer_groups USING btree (is_active) WHERE (is_active = true);

CREATE INDEX IF NOT EXISTS idx_customer_groups_name ON customer_groups USING btree (name);

CREATE INDEX IF NOT EXISTS idx_customers_customer_group ON customers USING btree (customer_group_id);

CREATE INDEX IF NOT EXISTS idx_customers_is_active ON customers USING btree (is_active);

CREATE INDEX IF NOT EXISTS idx_customers_name_trgm ON customers USING gin (name public.gin_trgm_ops);

CREATE INDEX IF NOT EXISTS idx_customers_phone ON customers USING btree (phone);

CREATE INDEX IF NOT EXISTS idx_customers_store_id ON customers USING btree (store_id);

CREATE INDEX IF NOT EXISTS idx_dead_letter_events_created_at ON dead_letter_events USING btree (created_at);

CREATE INDEX IF NOT EXISTS idx_dead_letter_events_event_type ON dead_letter_events USING btree (event_type);

CREATE INDEX IF NOT EXISTS idx_goods_receipt_items_gr ON goods_receipt_items USING btree (goods_receipt_id);

CREATE INDEX IF NOT EXISTS idx_goods_receipts_po ON goods_receipts USING btree (purchase_order_id);

CREATE INDEX IF NOT EXISTS idx_ia_created ON inventory_adjustments USING btree (created_at DESC);

CREATE INDEX IF NOT EXISTS idx_ia_items_adj ON inventory_adjustment_items USING btree (adjustment_id);

CREATE INDEX IF NOT EXISTS idx_ia_items_product ON inventory_adjustment_items USING btree (product_id);

CREATE INDEX IF NOT EXISTS idx_ia_session ON inventory_adjustments USING btree (session_id);

CREATE INDEX IF NOT EXISTS idx_import_errors_job ON import_errors USING btree (import_job_id);

CREATE INDEX IF NOT EXISTS idx_import_jobs_module ON import_jobs USING btree (module);

CREATE INDEX IF NOT EXISTS idx_import_jobs_status ON import_jobs USING btree (status);

CREATE INDEX IF NOT EXISTS idx_import_jobs_user ON import_jobs USING btree (user_id);

CREATE INDEX IF NOT EXISTS idx_import_rows_job ON import_rows USING btree (import_job_id);

CREATE INDEX IF NOT EXISTS idx_inventory_movements_product ON inventory_movements USING btree (product_id);

CREATE INDEX IF NOT EXISTS idx_payment_methods_code ON payment_methods USING btree (code);

CREATE INDEX IF NOT EXISTS idx_payment_methods_is_active ON payment_methods USING btree (is_active);

CREATE INDEX IF NOT EXISTS idx_pricing_rules_active ON pricing_rules USING btree (product_id, is_active) WHERE (is_active = true);

CREATE INDEX IF NOT EXISTS idx_pricing_rules_brand ON pricing_rules USING btree (brand_id) WHERE (brand_id IS NOT NULL);

CREATE INDEX IF NOT EXISTS idx_pricing_rules_category ON pricing_rules USING btree (category_id) WHERE (category_id IS NOT NULL);

CREATE INDEX IF NOT EXISTS idx_pricing_rules_effective ON pricing_rules USING btree (effective_from, effective_until) WHERE (is_active = true);

CREATE INDEX IF NOT EXISTS idx_pricing_rules_group ON pricing_rules USING btree (customer_group_id) WHERE (customer_group_id IS NOT NULL);

CREATE INDEX IF NOT EXISTS idx_pricing_rules_method ON pricing_rules USING btree (pricing_method);

CREATE INDEX IF NOT EXISTS idx_pricing_rules_product_id ON pricing_rules USING btree (product_id);

CREATE INDEX IF NOT EXISTS idx_pricing_rules_status ON pricing_rules USING btree (status);

CREATE INDEX IF NOT EXISTS idx_pricing_rules_store ON pricing_rules USING btree (store_id) WHERE (store_id IS NOT NULL);

CREATE INDEX IF NOT EXISTS idx_pricing_rules_type ON pricing_rules USING btree (pricing_type);

CREATE INDEX IF NOT EXISTS idx_product_stock_location_id ON product_stock USING btree (location_id);

CREATE INDEX IF NOT EXISTS idx_product_stock_product_id ON product_stock USING btree (product_id);

CREATE INDEX IF NOT EXISTS idx_product_stock_store_id ON product_stock USING btree (store_id);

CREATE INDEX IF NOT EXISTS idx_product_stock_warehouse_id ON product_stock USING btree (warehouse_id);

CREATE UNIQUE INDEX IF NOT EXISTS idx_product_suppliers_one_preferred ON product_suppliers USING btree (product_id) WHERE (is_preferred = true);

CREATE INDEX IF NOT EXISTS idx_product_suppliers_product ON product_suppliers USING btree (product_id);

CREATE INDEX IF NOT EXISTS idx_product_suppliers_supplier ON product_suppliers USING btree (supplier_id);

CREATE INDEX IF NOT EXISTS idx_products_barcode ON products USING btree (barcode);

CREATE INDEX IF NOT EXISTS idx_products_brand ON products USING btree (brand_id);

CREATE INDEX IF NOT EXISTS idx_products_category ON products USING btree (category_id);

CREATE INDEX IF NOT EXISTS idx_products_category_active ON products USING btree (category_id) WHERE (deleted_at IS NULL);

CREATE INDEX IF NOT EXISTS idx_products_name_trgm ON products USING gin (name public.gin_trgm_ops);

CREATE INDEX IF NOT EXISTS idx_products_search_vector ON products USING gin (search_vector);

CREATE INDEX IF NOT EXISTS idx_products_sku ON products USING btree (sku);

CREATE INDEX IF NOT EXISTS idx_products_store ON products USING btree (store_id);

CREATE INDEX IF NOT EXISTS idx_products_tax_class ON products USING btree (tax_class_id);

CREATE UNIQUE INDEX IF NOT EXISTS idx_products_unique_active_barcode ON products USING btree (barcode) WHERE (deleted_at IS NULL);

CREATE INDEX IF NOT EXISTS idx_products_uom ON products USING btree (unit_of_measure_id);

CREATE INDEX IF NOT EXISTS idx_purchase_order_items_po ON purchase_order_items USING btree (purchase_order_id);

CREATE INDEX IF NOT EXISTS idx_purchase_orders_created_at ON purchase_orders USING btree (created_at DESC);

CREATE INDEX IF NOT EXISTS idx_purchase_orders_status ON purchase_orders USING btree (status);

CREATE INDEX IF NOT EXISTS idx_purchase_orders_status_store ON purchase_orders USING btree (status, store_id);

CREATE INDEX IF NOT EXISTS idx_purchase_orders_store ON purchase_orders USING btree (store_id);

CREATE INDEX IF NOT EXISTS idx_purchase_orders_supplier ON purchase_orders USING btree (supplier_id);

CREATE INDEX IF NOT EXISTS idx_sale_items_pricing_type ON sale_items USING btree (pricing_type);

CREATE INDEX IF NOT EXISTS idx_sale_items_product ON sale_items USING btree (product_id);

CREATE INDEX IF NOT EXISTS idx_sale_items_sale ON sale_items USING btree (sale_id);

CREATE INDEX IF NOT EXISTS idx_sale_items_sale_id ON sale_items USING btree (sale_id) INCLUDE (product_id, quantity, unit_price, subtotal);

CREATE INDEX IF NOT EXISTS idx_sale_payments_method ON sale_payments USING btree (payment_method_id);

CREATE INDEX IF NOT EXISTS idx_sale_payments_sale ON sale_payments USING btree (sale_id);

CREATE INDEX IF NOT EXISTS idx_sales_active_aggregations ON sales USING btree (created_at DESC) WHERE ((status)::text = 'completed'::text);

CREATE INDEX IF NOT EXISTS idx_sales_aggregation ON sales USING btree (store_id, created_at DESC, total_amount) INCLUDE (id, invoice_number, cashier_id, status);

CREATE INDEX IF NOT EXISTS idx_sales_cashier ON sales USING btree (cashier_id);

CREATE INDEX IF NOT EXISTS idx_sales_created ON sales USING btree (created_at);

CREATE INDEX IF NOT EXISTS idx_sales_invoice_number_trgm ON sales USING gin (invoice_number public.gin_trgm_ops);

CREATE INDEX IF NOT EXISTS idx_sales_shift_id ON sales USING btree (shift_id);

CREATE INDEX IF NOT EXISTS idx_sales_shift_status ON sales USING btree (shift_id, status);

CREATE INDEX IF NOT EXISTS idx_sales_status_created_store ON sales USING btree (status, created_at, store_id) INCLUDE (total_amount);

CREATE INDEX IF NOT EXISTS idx_sales_store ON sales USING btree (store_id);

CREATE INDEX IF NOT EXISTS idx_shifts_opened_at ON shifts USING btree (opened_at);

CREATE INDEX IF NOT EXISTS idx_shifts_status ON shifts USING btree (status);

CREATE INDEX IF NOT EXISTS idx_shifts_store_id ON shifts USING btree (store_id);

CREATE INDEX IF NOT EXISTS idx_shifts_user_id ON shifts USING btree (user_id);

CREATE INDEX IF NOT EXISTS idx_so_items_opname_status ON stock_opname_items USING btree (stock_opname_id, status);

CREATE INDEX IF NOT EXISTS idx_so_recounts_opname ON stock_opname_recount_requests USING btree (stock_opname_id);

CREATE INDEX IF NOT EXISTS idx_so_scopes_opname ON stock_opname_session_scopes USING btree (stock_opname_id);

CREATE INDEX IF NOT EXISTS idx_so_scopes_type_id ON stock_opname_session_scopes USING btree (scope_type, scope_id);

CREATE INDEX IF NOT EXISTS idx_stock_opname_assignments_opname ON stock_opname_assignments USING btree (stock_opname_id);

CREATE INDEX IF NOT EXISTS idx_stock_opname_assignments_user ON stock_opname_assignments USING btree (user_id);

CREATE INDEX IF NOT EXISTS idx_stock_opname_counts_counted ON stock_opname_counts USING btree (counted_at);

CREATE INDEX IF NOT EXISTS idx_stock_opname_counts_item ON stock_opname_counts USING btree (stock_opname_item_id);

CREATE INDEX IF NOT EXISTS idx_stock_opname_created ON stock_opnames USING btree (created_at DESC);

CREATE INDEX IF NOT EXISTS idx_stock_opname_items_opname ON stock_opname_items USING btree (stock_opname_id);

CREATE INDEX IF NOT EXISTS idx_stock_opname_items_product ON stock_opname_items USING btree (product_id);

CREATE INDEX IF NOT EXISTS idx_stock_opname_items_status ON stock_opname_items USING btree (status);

CREATE INDEX IF NOT EXISTS idx_stock_opname_scope ON stock_opnames USING btree (scope_type, scope_id);

CREATE INDEX IF NOT EXISTS idx_stock_opname_status ON stock_opnames USING btree (status);

CREATE INDEX IF NOT EXISTS idx_stock_opname_status_created ON stock_opnames USING btree (status, created_at);

CREATE INDEX IF NOT EXISTS idx_stock_opnames_location_id ON stock_opnames USING btree (location_id);

CREATE INDEX IF NOT EXISTS idx_stock_opnames_store_id ON stock_opnames USING btree (store_id);

CREATE INDEX IF NOT EXISTS idx_storage_locations_active ON storage_locations USING btree (is_active);

CREATE INDEX IF NOT EXISTS idx_storage_locations_store_id ON storage_locations USING btree (store_id);

CREATE INDEX IF NOT EXISTS idx_storage_locations_warehouse_id ON storage_locations USING btree (warehouse_id);

CREATE INDEX IF NOT EXISTS idx_suppliers_code ON suppliers USING btree (code);

CREATE INDEX IF NOT EXISTS idx_users_reports_to ON users USING btree (reports_to);

CREATE INDEX IF NOT EXISTS idx_users_role ON users USING btree (role_id);

CREATE INDEX IF NOT EXISTS idx_users_store ON users USING btree (store_id);

CREATE UNIQUE INDEX IF NOT EXISTS uq_cart_sessions_open_cashier ON cart_sessions USING btree (cashier_id) WHERE ((status)::text = 'open'::text);

CREATE UNIQUE INDEX IF NOT EXISTS uq_open_shift_per_user ON shifts USING btree (user_id) WHERE ((status)::text = 'open'::text);

CREATE UNIQUE INDEX IF NOT EXISTS uq_stock_opname_assignment ON stock_opname_assignments USING btree (stock_opname_id, user_id, role);

-- ============================================================
-- View
-- ============================================================
CREATE OR REPLACE VIEW v_products_full AS
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
    ps_preferred.supplier_name
   FROM ((((((public.products p
     LEFT JOIN categories c ON ((p.category_id = c.id)))
     LEFT JOIN brands b ON ((p.brand_id = b.id)))
     LEFT JOIN units_of_measure u ON ((p.unit_of_measure_id = u.id)))
     LEFT JOIN LATERAL ( SELECT product_stock.quantity
           FROM product_stock
          WHERE (product_stock.product_id = p.id)
          ORDER BY ((product_stock.warehouse_id IS NULL) AND (product_stock.store_id IS NULL)) DESC
         LIMIT 1) ps ON (true))
     LEFT JOIN tax_classes tc ON ((tc.id = p.tax_class_id)))
     LEFT JOIN LATERAL ( SELECT s.id AS supplier_id,
            s.name AS supplier_name
           FROM (public.product_suppliers ps_1
             JOIN suppliers s ON (((ps_1.supplier_id = s.id) AND (s.deleted_at IS NULL))))
          WHERE ((ps_1.product_id = p.id) AND (ps_1.is_preferred = true))
         LIMIT 1) ps_preferred ON (true))
  WHERE (p.deleted_at IS NULL);

-- ============================================================
-- Materialized views
-- ============================================================
CREATE MATERIALIZED VIEW IF NOT EXISTS public.mv_daily_sales AS
 SELECT date((created_at AT TIME ZONE 'Asia/Jakarta'::text)) AS sale_date,
    store_id,
    count(*) AS transaction_count,
    count(DISTINCT cashier_id) AS active_cashiers,
    COALESCE(sum(total_amount), (0)::bigint) AS total_revenue,
    COALESCE(sum(subtotal), (0)::bigint) AS total_subtotal,
    COALESCE(sum(discount), (0)::bigint) AS total_discount,
    COALESCE(sum(tax), (0)::bigint) AS total_tax
   FROM sales
  WHERE ((status)::text = 'completed'::text)
  GROUP BY (date((created_at AT TIME ZONE 'Asia/Jakarta'::text))), store_id
  WITH DATA;

CREATE MATERIALIZED VIEW IF NOT EXISTS public.mv_dashboard_totals AS
 SELECT store_id,
    count(*) AS transaction_count,
    COALESCE(sum(total_amount), (0)::bigint) AS total_revenue
   FROM sales
  WHERE ((status)::text = 'completed'::text)
  GROUP BY store_id
  WITH DATA;

CREATE MATERIALIZED VIEW IF NOT EXISTS public.mv_hourly_sales AS
 SELECT date_trunc('hour'::text, (created_at AT TIME ZONE 'Asia/Jakarta'::text)) AS sale_hour,
    store_id,
    count(*) AS transaction_count,
    COALESCE(sum(total_amount), (0)::bigint) AS total_revenue
   FROM sales
  WHERE ((status)::text = 'completed'::text)
  GROUP BY (date_trunc('hour'::text, (created_at AT TIME ZONE 'Asia/Jakarta'::text))), store_id
  WITH DATA;

CREATE UNIQUE INDEX IF NOT EXISTS idx_mv_daily_sales_date_store ON mv_daily_sales USING btree (sale_date, store_id);

CREATE UNIQUE INDEX IF NOT EXISTS idx_mv_dashboard_totals_store ON mv_dashboard_totals USING btree (store_id);

CREATE UNIQUE INDEX IF NOT EXISTS idx_mv_hourly_sales_hour_store ON mv_hourly_sales USING btree (sale_hour, store_id);

-- ============================================================
-- Trigger
-- ============================================================
DROP TRIGGER IF EXISTS trg_products_search_vector ON products;
CREATE TRIGGER trg_products_search_vector BEFORE INSERT OR UPDATE OF name, sku, barcode ON products FOR EACH ROW EXECUTE FUNCTION public.products_search_vector_update();

COMMIT;

BEGIN;

-- Roles
INSERT INTO roles (name, description, is_system) VALUES
    ('superadmin', 'Super Administrator', True),
    ('admin', 'Administrator', True),
    ('manager', 'Manager / Kepala Toko', True),
    ('cashier', 'Kasir', True),
    ('staff', 'Staff Gudang', True)
ON CONFLICT (name) DO NOTHING;

-- Payment methods
INSERT INTO payment_methods (code, name, is_active, requires_reference, sort_order) VALUES
    ('CASH', 'Cash', True, False, 1),
    ('CARD', 'Card', True, True, 2),
    ('E_WALLET', 'E-Wallet', True, True, 3),
    ('TRANSFER', 'Transfer', True, True, 4),
    ('QRIS', 'QRIS', True, False, 5)
ON CONFLICT (code) DO NOTHING;

-- Customer groups
INSERT INTO customer_groups (name, description, is_active, color) VALUES
    ('Walk-in', 'Pelanggan umum tanpa kartu member', True, '#636E72'),
    ('Member', 'Pelanggan terdaftar dengan kartu member', True, '#00B894'),
    ('VIP', 'Pelanggan prioritas dengan harga khusus', True, '#6C5CE7')
ON CONFLICT (name) DO NOTHING;

-- Walk-in customer
INSERT INTO customers (id, name, phone, email, is_walk_in, is_active, store_id, customer_group_id)
VALUES (1, 'Pelanggan Umum / Walk-in', '0000000000', 'walk-in@retail-pos.local', True, True, 1, NULL)
ON CONFLICT (id) DO NOTHING;

-- Permissions
INSERT INTO permissions (code, name, description) VALUES
    ('audit.view', 'Lihat Log Audit', 'Bisa melihat riwayat log audit'),
    ('category.create', 'Tambah Kategori', 'Bisa menambah kategori'),
    ('category.delete', 'Hapus Kategori', 'Bisa menghapus kategori'),
    ('category.export', 'Export Kategori', 'Bisa mengexport data kategori'),
    ('category.import', 'Import Kategori', 'Bisa mengimport data kategori'),
    ('category.update', 'Edit Kategori', 'Bisa mengubah kategori'),
    ('category.view', 'Lihat Kategori', 'Bisa melihat kategori'),
    ('customer.create', 'Create Customer', 'Add new customers'),
    ('customer.delete', 'Delete Customer', 'Deactivate/delete customers'),
    ('customer.export', 'Export Pelanggan', 'Bisa mengexport data pelanggan'),
    ('customer.import', 'Import Pelanggan', 'Bisa mengimport data pelanggan'),
    ('customer.update', 'Update Customer', 'Edit customer information'),
    ('customer.view', 'View Customers', 'View customer list and details'),
    ('customer_group.create', 'Buat Data Customer Group', NULL),
    ('customer_group.delete', 'Hapus Data Customer Group', NULL),
    ('customer_group.update', 'Edit Data Customer Group', NULL),
    ('customer_group.view', 'Lihat Data Customer Group', NULL),
    ('dashboard.view', 'Lihat Dashboard', 'Bisa melihat dashboard utama'),
    ('inventory.adjust', 'Adjust Inventory', 'Manual stock adjustment'),
    ('pricing.create', 'Tambah Aturan Harga', 'Bisa menambah aturan harga baru'),
    ('pricing.delete', 'Hapus Aturan Harga', 'Bisa menghapus aturan harga'),
    ('pricing.update', 'Edit Aturan Harga', 'Bisa mengubah aturan harga'),
    ('pricing.view', 'Lihat Aturan Harga', 'Bisa melihat daftar aturan harga'),
    ('product.cost.view', 'Product Cost View', 'View sensitive cost data (cost, margin, purchase price, markup, profit)'),
    ('product.create', 'Tambah Produk', 'Bisa menambah produk baru'),
    ('product.delete', 'Hapus Produk', 'Bisa menghapus produk'),
    ('product.export', 'Export Produk', 'Bisa mengexport data produk'),
    ('product.history.view', 'Product History View', 'View product entity history (audit trail: created & updated timestamps)'),
    ('product.import', 'Import Produk', 'Bisa mengimport data produk'),
    ('product.update', 'Edit Produk', 'Bisa mengubah data produk'),
    ('product.view', 'Lihat Produk', 'Bisa melihat daftar produk'),
    ('purchase_order.cancel', 'Batalkan Purchase Order', 'Bisa membatalkan purchase order (draft/confirmed)'),
    ('purchase_order.confirm', 'Konfirmasi Purchase Order', 'Bisa mengkonfirmasi purchase order'),
    ('purchase_order.create', 'Buat Purchase Order', 'Bisa membuat purchase order baru'),
    ('purchase_order.delete', 'Hapus Purchase Order', 'Bisa menghapus purchase order'),
    ('purchase_order.receive', 'Terima Barang', 'Bisa membuat goods receipt untuk purchase order'),
    ('purchase_order.update', 'Edit Purchase Order', 'Bisa mengubah purchase order'),
    ('purchase_order.view', 'Lihat Purchase Order', 'Bisa melihat daftar dan detail purchase order'),
    ('report.view', 'Lihat Laporan', 'Bisa melihat laporan keuangan & stok'),
    ('role.create', 'Tambah Role', 'Bisa menambah role baru'),
    ('role.delete', 'Hapus Role', 'Bisa menghapus role'),
    ('role.update', 'Edit Role', 'Bisa mengubah role'),
    ('role.view', 'Lihat Role', 'Bisa melihat daftar role'),
    ('sale.create', 'Buat Penjualan', 'Bisa membuat transaksi penjualan'),
    ('sale.park', 'Parkir Penjualan', 'Bisa menyimpan sementara, mengambil kembali, dan membatalkan penjualan yang diparkir'),
    ('sale.view', 'Lihat Penjualan', 'Bisa melihat daftar penjualan'),
    ('shift.audit', 'Audit shift', 'Audit fisik cash shift'),
    ('shift.create', 'Kelola shift', 'Buka dan tutup shift'),
    ('shift.review', 'Review shift', 'Review dan setujui selisih shift'),
    ('shift.view', 'Baca shift', 'Lihat daftar dan detail shift'),
    ('stock_opname.assign', 'Atur Petugas Stock Opname', 'Bisa menugaskan counter dan supervisor'),
    ('stock_opname.cancel', 'Batalkan Stock Opname', 'Bisa membatalkan sesi stock opname'),
    ('stock_opname.close', 'Tutup Stock Opname', 'Bisa menutup sesi stock opname'),
    ('stock_opname.count', 'Hitung Stok Fisik', 'Bisa melakukan penghitungan stok fisik'),
    ('stock_opname.create', 'Buat Stock Opname', 'Bisa membuat sesi stock opname baru'),
    ('stock_opname.export', 'Ekspor Stock Opname', 'Bisa mengekspor laporan stock opname'),
    ('stock_opname.post', 'Posting Stock Opname', 'Bisa memposting penyesuaian stok'),
    ('stock_opname.recount', 'Minta Hitung Ulang', 'Bisa meminta penghitungan ulang'),
    ('stock_opname.report', 'Laporan Stock Opname', 'Bisa melihat laporan stock opname'),
    ('stock_opname.submit', 'Submit Stock Opname', 'Bisa mengirim hasil penghitungan'),
    ('stock_opname.verify', 'Verifikasi Stock Opname', 'Bisa memverifikasi hasil stock opname'),
    ('stock_opname.view', 'Lihat Stock Opname', 'Bisa melihat daftar dan detail stock opname'),
    ('storage_location.create', 'Buat Lokasi Penyimpanan', 'Bisa membuat lokasi penyimpanan baru'),
    ('storage_location.delete', 'Hapus Lokasi Penyimpanan', 'Bisa menghapus lokasi penyimpanan'),
    ('storage_location.update', 'Ubah Lokasi Penyimpanan', 'Bisa mengubah lokasi penyimpanan'),
    ('storage_location.view', 'Lihat Lokasi Penyimpanan', 'Bisa melihat daftar dan detail lokasi penyimpanan'),
    ('store.create', 'Buat Data Toko', NULL),
    ('store.delete', 'Hapus Data Toko', NULL),
    ('store.update', 'Edit Data Toko', NULL),
    ('store.view', 'Lihat Data Toko', NULL),
    ('user.create', 'Tambah Pengguna', 'Bisa menambah pengguna baru'),
    ('user.delete', 'Hapus Pengguna', 'Bisa menghapus pengguna'),
    ('user.update', 'Edit Pengguna', 'Bisa mengubah data pengguna'),
    ('user.view', 'Lihat Pengguna', 'Bisa melihat daftar pengguna')
ON CONFLICT (code) DO NOTHING;

-- Superadmin
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM roles r, permissions p WHERE r.name = 'superadmin'
    AND p.code IN (
        'audit.view',
        'category.create',
        'category.delete',
        'category.export',
        'category.import',
        'category.update',
        'category.view',
        'customer.create',
        'customer.delete',
        'customer.export',
        'customer.import',
        'customer.update',
        'customer.view',
        'customer_group.create',
        'customer_group.delete',
        'customer_group.update',
        'customer_group.view',
        'dashboard.view',
        'inventory.adjust',
        'pricing.create',
        'pricing.delete',
        'pricing.update',
        'pricing.view',
        'product.cost.view',
        'product.create',
        'product.delete',
        'product.export',
        'product.history.view',
        'product.import',
        'product.update',
        'product.view',
        'purchase_order.cancel',
        'purchase_order.confirm',
        'purchase_order.create',
        'purchase_order.delete',
        'purchase_order.receive',
        'purchase_order.update',
        'purchase_order.view',
        'report.view',
        'role.create',
        'role.delete',
        'role.update',
        'role.view',
        'sale.create',
        'sale.park',
        'sale.view',
        'shift.audit',
        'shift.create',
        'shift.review',
        'shift.view',
        'stock_opname.assign',
        'stock_opname.cancel',
        'stock_opname.close',
        'stock_opname.count',
        'stock_opname.create',
        'stock_opname.export',
        'stock_opname.post',
        'stock_opname.recount',
        'stock_opname.report',
        'stock_opname.submit',
        'stock_opname.verify',
        'stock_opname.view',
        'storage_location.create',
        'storage_location.delete',
        'storage_location.update',
        'storage_location.view',
        'store.create',
        'store.delete',
        'store.update',
        'store.view',
        'user.create',
        'user.delete',
        'user.update',
        'user.view'
    )
ON CONFLICT DO NOTHING;

-- Admin
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM roles r, permissions p WHERE r.name = 'admin'
    AND p.code IN (
        'category.create',
        'category.delete',
        'category.export',
        'category.import',
        'category.update',
        'category.view',
        'customer.create',
        'customer.delete',
        'customer.export',
        'customer.import',
        'customer.update',
        'customer.view',
        'customer_group.create',
        'customer_group.delete',
        'customer_group.update',
        'customer_group.view',
        'dashboard.view',
        'inventory.adjust',
        'pricing.create',
        'pricing.delete',
        'pricing.update',
        'pricing.view',
        'product.cost.view',
        'product.create',
        'product.delete',
        'product.export',
        'product.history.view',
        'product.import',
        'product.update',
        'product.view',
        'purchase_order.cancel',
        'purchase_order.confirm',
        'purchase_order.create',
        'purchase_order.receive',
        'purchase_order.update',
        'purchase_order.view',
        'report.view',
        'role.create',
        'role.view',
        'sale.create',
        'sale.park',
        'sale.view',
        'shift.audit',
        'shift.create',
        'shift.review',
        'shift.view',
        'stock_opname.assign',
        'stock_opname.cancel',
        'stock_opname.close',
        'stock_opname.count',
        'stock_opname.create',
        'stock_opname.export',
        'stock_opname.post',
        'stock_opname.recount',
        'stock_opname.report',
        'stock_opname.submit',
        'stock_opname.verify',
        'stock_opname.view',
        'storage_location.create',
        'storage_location.delete',
        'storage_location.update',
        'storage_location.view',
        'store.create',
        'store.delete',
        'store.update',
        'store.view',
        'user.create',
        'user.update',
        'user.view'
    )
ON CONFLICT DO NOTHING;

-- Manager
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM roles r, permissions p WHERE r.name = 'manager'
    AND p.code IN (
        'category.create',
        'category.view',
        'customer.create',
        'customer.update',
        'customer.view',
        'customer_group.view',
        'dashboard.view',
        'inventory.adjust',
        'pricing.create',
        'pricing.update',
        'pricing.view',
        'product.cost.view',
        'product.update',
        'product.view',
        'purchase_order.cancel',
        'purchase_order.confirm',
        'purchase_order.create',
        'purchase_order.receive',
        'purchase_order.update',
        'purchase_order.view',
        'report.view',
        'sale.park',
        'sale.view',
        'shift.audit',
        'shift.create',
        'shift.review',
        'shift.view',
        'stock_opname.assign',
        'stock_opname.cancel',
        'stock_opname.close',
        'stock_opname.create',
        'stock_opname.export',
        'stock_opname.post',
        'stock_opname.recount',
        'stock_opname.report',
        'stock_opname.verify',
        'stock_opname.view',
        'storage_location.view'
    )
ON CONFLICT DO NOTHING;

-- Cashier
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM roles r, permissions p WHERE r.name = 'cashier'
    AND p.code IN (
        'customer.view',
        'product.view',
        'sale.create',
        'sale.park',
        'sale.view',
        'shift.create',
        'shift.view',
        'stock_opname.count',
        'stock_opname.submit',
        'stock_opname.view',
        'storage_location.view'
    )
ON CONFLICT DO NOTHING;

-- Staff
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM roles r, permissions p WHERE r.name = 'staff'
    AND p.code IN (
        'product.view',
        'stock_opname.count',
        'stock_opname.submit',
        'stock_opname.view',
        'storage_location.view'
    )
ON CONFLICT DO NOTHING;

-- Default users (password: admin123)
INSERT INTO users (username, email, password_hash, role_id, is_active)
SELECT 'superadmin', 'superadmin@retailpos.local', crypt('admin123', gen_salt('bf', 14)), r.id, true
FROM roles r WHERE r.name = 'superadmin'
ON CONFLICT (username) DO NOTHING;

INSERT INTO users (username, email, password_hash, role_id, is_active)
SELECT 'admin', 'admin@retailpos.local', crypt('admin123', gen_salt('bf', 14)), r.id, true
FROM roles r WHERE r.name = 'admin'
ON CONFLICT (username) DO NOTHING;

INSERT INTO users (username, email, password_hash, role_id, is_active)
SELECT 'manager', 'manager@retailpos.local', crypt('admin123', gen_salt('bf', 14)), r.id, true
FROM roles r WHERE r.name = 'manager'
ON CONFLICT (username) DO NOTHING;

INSERT INTO users (username, email, password_hash, role_id, is_active)
SELECT 'cashier', 'cashier@retailpos.local', crypt('admin123', gen_salt('bf', 14)), r.id, true
FROM roles r WHERE r.name = 'cashier'
ON CONFLICT (username) DO NOTHING;

INSERT INTO users (username, email, password_hash, role_id, reports_to, is_active)
SELECT 'staff', 'staff@retailpos.local', crypt('admin123', gen_salt('bf', 14)), r.id,
       (SELECT id FROM users WHERE username = 'manager'), true
FROM roles r WHERE r.name = 'staff'
ON CONFLICT (username) DO NOTHING;

COMMIT;

