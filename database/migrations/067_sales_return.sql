-- 067_sales_return.sql
-- Sales Return (Retur Penjualan) — PRD docs/prd/PRD-Sales-Return.md.
--
-- Five new tables owned by the salereturn module (internal/salereturn):
-- sale_returns (header), sale_return_items (per sale_item line), 
-- sale_return_replacements (replacement deliveries for mixed returns),
-- sale_return_payments (cash/non-cash refund split) and return_approvals
-- (manager approval workflow, PRD §6).
--
-- Design decisions from the plan review (docs/design/sales-return-plan.md
-- "Review amendments"):
--   * Returns never restock: a damaged returned unit is written off and
--     inventory movements are NOT written by Phase 1 (DeductStock for
--     replacements is a pure product_stock decrement). Hence no movement
--     rows here.
--   * sale_returns.change_due snapshots the original sale's change at
--     processing time so the cash refund is exactly Σ(cash payments) − change
--     due (sale_payments has no change column; sales does).
--   * Consignment-owned lines are rejected by the service (a follow-up will
--     add consignment restock); nothing in this schema touches
--     consignment_stock.
--   * Returns adjust shift running totals via shift.TotalUpdater
--     (ApplyReturnAdjustment); nothing in this schema touches shifts.
--
-- Permanently re-runnable per the migration replay contract: every statement
-- is guarded (IF NOT EXISTS / WHERE NOT EXISTS / ON CONFLICT DO NOTHING). The
-- permission rows and role grants are mirrored in 000_baseline.sql
-- (fresh-install parity, per 065/066); internal/permissions/baseline_test.go
-- keeps both in step.
--
-- The permission ids (98, 99) only matter as unique keys here; the baseline
-- seeds by id and both files insert ON CONFLICT DO NOTHING.

-- ---------------------------------------------------------------------------
-- Permissions sale.return / sale.return.approve + grants
-- ---------------------------------------------------------------------------
-- sale.return      : cashier, manager, superadmin (a cashier processes the
--                    customer-facing return; supervisor deliberately excluded
--                    per PRD §14 role matrix — flagged deviation: PRD "admin"
--                    does not exist as a role, mapped to manager).
-- sale.return.approve : manager, superadmin.

DO $$
BEGIN
  INSERT INTO public.permissions (id, code, name, description)
  VALUES (98, 'sale.return', 'Retur Penjualan', 'Memproses retur barang yang dibeli pelanggan')
  ON CONFLICT DO NOTHING;

  INSERT INTO public.permissions (id, code, name, description)
  VALUES (99, 'sale.return.approve', 'Persetujuan Retur', 'Menyetujui retur dengan nilai di atas batas persetujuan')
  ON CONFLICT DO NOTHING;

  INSERT INTO role_permissions (role_id, permission_id)
  SELECT r.id, p.id
  FROM roles r
  JOIN permissions p ON p.code IN ('sale.return')
  WHERE r.name = 'cashier'
    AND NOT EXISTS (
      SELECT 1 FROM role_permissions rp
      WHERE rp.role_id = r.id AND rp.permission_id = p.id
    );

  INSERT INTO role_permissions (role_id, permission_id)
  SELECT r.id, p.id
  FROM roles r
  JOIN permissions p ON p.code IN ('sale.return', 'sale.return.approve')
  WHERE r.name = 'manager'
    AND NOT EXISTS (
      SELECT 1 FROM role_permissions rp
      WHERE rp.role_id = r.id AND rp.permission_id = p.id
    );

  INSERT INTO role_permissions (role_id, permission_id)
  SELECT r.id, p.id
  FROM roles r
  JOIN permissions p ON p.code IN ('sale.return', 'sale.return.approve')
  WHERE r.name = 'superadmin'
    AND NOT EXISTS (
      SELECT 1 FROM role_permissions rp
      WHERE rp.role_id = r.id AND rp.permission_id = p.id
    );
END;
$$;

-- ---------------------------------------------------------------------------
-- App settings (policy defaults, PRD §5; key has no description column)
-- ---------------------------------------------------------------------------

INSERT INTO public.app_settings ("key", "value")
VALUES ('return.window_days', '1')
ON CONFLICT DO NOTHING;
INSERT INTO public.app_settings ("key", "value")
VALUES ('return.approval_threshold', '500000')
ON CONFLICT DO NOTHING;
INSERT INTO public.app_settings ("key", "value")
VALUES ('return.approval_timeout_minutes', '5')
ON CONFLICT DO NOTHING;

-- ---------------------------------------------------------------------------
-- Sequences
-- ---------------------------------------------------------------------------

CREATE SEQUENCE IF NOT EXISTS public.return_seq AS integer START WITH 1 INCREMENT BY 1 NO CYCLE;

-- Per-table id sequences (the tables declare bare integer ids; the defaults
-- are attached at the end of this file).
CREATE SEQUENCE IF NOT EXISTS public.sale_returns_id_seq AS integer START WITH 1 INCREMENT BY 1 NO CYCLE;
CREATE SEQUENCE IF NOT EXISTS public.sale_return_items_id_seq AS integer START WITH 1 INCREMENT BY 1 NO CYCLE;
CREATE SEQUENCE IF NOT EXISTS public.sale_return_replacements_id_seq AS integer START WITH 1 INCREMENT BY 1 NO CYCLE;
CREATE SEQUENCE IF NOT EXISTS public.sale_return_payments_id_seq AS integer START WITH 1 INCREMENT BY 1 NO CYCLE;
CREATE SEQUENCE IF NOT EXISTS public.return_approvals_id_seq AS integer START WITH 1 INCREMENT BY 1 NO CYCLE;

-- ---------------------------------------------------------------------------
-- sale_returns (header, PRD §8.1)
-- ---------------------------------------------------------------------------

CREATE TABLE IF NOT EXISTS public.sale_returns (
    id integer NOT NULL,
    return_number character varying(30) NOT NULL,
    original_sale_id integer NOT NULL,
    cashier_id integer NOT NULL,
    store_id integer NOT NULL,
    customer_id integer,
    shift_id integer NOT NULL,
    return_type character varying(20) NOT NULL,
    reason text NOT NULL,
    condition character varying(20) NOT NULL,
    notes text,
    subtotal integer DEFAULT 0 NOT NULL,
    discount integer DEFAULT 0 NOT NULL,
    restocking_fee integer DEFAULT 0 NOT NULL,
    tax_reversal integer DEFAULT 0 NOT NULL,
    total_refund_amount integer DEFAULT 0 NOT NULL,
    status character varying(20) DEFAULT 'pending_approval' NOT NULL,
    change_due integer DEFAULT 0 NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    expires_at timestamp with time zone,
    CONSTRAINT sale_returns_pkey PRIMARY KEY (id),
    CONSTRAINT sale_returns_return_number_key UNIQUE (return_number),
    CONSTRAINT chk_sale_return_type CHECK (((return_type)::text = ANY (ARRAY[('replacement'::character varying)::text, ('refund'::character varying)::text]))),
    CONSTRAINT chk_sale_return_condition CHECK (((condition)::text = ANY (ARRAY[('damaged'::character varying)::text, ('functional'::character varying)::text]))),
    CONSTRAINT chk_sale_return_status CHECK (((status)::text = ANY (ARRAY[('pending_approval'::character varying)::text, ('completed'::character varying)::text, ('cancelled'::character varying)::text, ('expired'::character varying)::text]))),
    CONSTRAINT chk_sale_return_total CHECK ((total_refund_amount >= 0))
);

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conrelid = 'public.sale_returns'::regclass AND conname = 'sale_returns_original_sale_id_fkey'
    ) THEN
        ALTER TABLE ONLY public.sale_returns ADD CONSTRAINT sale_returns_original_sale_id_fkey
            FOREIGN KEY (original_sale_id) REFERENCES public.sales(id) ON DELETE RESTRICT;
    END IF;
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conrelid = 'public.sale_returns'::regclass AND conname = 'sale_returns_cashier_id_fkey'
    ) THEN
        ALTER TABLE ONLY public.sale_returns ADD CONSTRAINT sale_returns_cashier_id_fkey
            FOREIGN KEY (cashier_id) REFERENCES public.users(id) ON DELETE RESTRICT;
    END IF;
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conrelid = 'public.sale_returns'::regclass AND conname = 'sale_returns_store_id_fkey'
    ) THEN
        ALTER TABLE ONLY public.sale_returns ADD CONSTRAINT sale_returns_store_id_fkey
            FOREIGN KEY (store_id) REFERENCES public.stores(id) ON DELETE RESTRICT;
    END IF;
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conrelid = 'public.sale_returns'::regclass AND conname = 'sale_returns_customer_id_fkey'
    ) THEN
        ALTER TABLE ONLY public.sale_returns ADD CONSTRAINT sale_returns_customer_id_fkey
            FOREIGN KEY (customer_id) REFERENCES public.customers(id) ON DELETE SET NULL;
    END IF;
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conrelid = 'public.sale_returns'::regclass AND conname = 'sale_returns_shift_id_fkey'
    ) THEN
        ALTER TABLE ONLY public.sale_returns ADD CONSTRAINT sale_returns_shift_id_fkey
            FOREIGN KEY (shift_id) REFERENCES public.shifts(id) ON DELETE RESTRICT;
    END IF;
END;
$$;

CREATE INDEX IF NOT EXISTS idx_sale_returns_original_sale ON public.sale_returns (original_sale_id);
CREATE INDEX IF NOT EXISTS idx_sale_returns_cashier ON public.sale_returns (cashier_id);
CREATE INDEX IF NOT EXISTS idx_sale_returns_store ON public.sale_returns (store_id);
CREATE INDEX IF NOT EXISTS idx_sale_returns_created ON public.sale_returns (created_at);
-- Pending approvals must be found by (status, expires_at) for the pending
-- queue and the lazy expiry filter.
CREATE INDEX IF NOT EXISTS idx_sale_returns_status ON public.sale_returns (status, expires_at);

-- ---------------------------------------------------------------------------
-- sale_return_items
-- ---------------------------------------------------------------------------

CREATE TABLE IF NOT EXISTS public.sale_return_items (
    id integer NOT NULL,
    sale_return_id integer NOT NULL,
    sale_item_id integer NOT NULL,
    product_id integer NOT NULL,
    product_name character varying(255) NOT NULL,
    quantity integer NOT NULL,
    unit_price integer NOT NULL,
    subtotal integer NOT NULL,
    dpp_amount integer DEFAULT 0 NOT NULL,
    tax_amount integer DEFAULT 0 NOT NULL,
    tax_rate numeric(5,2) DEFAULT 0 NOT NULL,
    condition character varying(20) NOT NULL,
    outcome character varying(20) NOT NULL,
    notes text,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT sale_return_items_pkey PRIMARY KEY (id),
    CONSTRAINT chk_sale_return_item_qty CHECK ((quantity > 0)),
    CONSTRAINT chk_sale_return_item_condition CHECK (((condition)::text = ANY (ARRAY[('damaged'::character varying)::text, ('functional'::character varying)::text]))),
    CONSTRAINT chk_sale_return_item_outcome CHECK (((outcome)::text = ANY (ARRAY[('restocked'::character varying)::text, ('written_off'::character varying)::text])))
);

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conrelid = 'public.sale_return_items'::regclass AND conname = 'sale_return_items_sale_return_id_fkey'
    ) THEN
        ALTER TABLE ONLY public.sale_return_items ADD CONSTRAINT sale_return_items_sale_return_id_fkey
            FOREIGN KEY (sale_return_id) REFERENCES public.sale_returns(id) ON DELETE CASCADE;
    END IF;
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conrelid = 'public.sale_return_items'::regclass AND conname = 'sale_return_items_sale_item_id_fkey'
    ) THEN
        ALTER TABLE ONLY public.sale_return_items ADD CONSTRAINT sale_return_items_sale_item_id_fkey
            FOREIGN KEY (sale_item_id) REFERENCES public.sale_items(id) ON DELETE RESTRICT;
    END IF;
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conrelid = 'public.sale_return_items'::regclass AND conname = 'sale_return_items_product_id_fkey'
    ) THEN
        ALTER TABLE ONLY public.sale_return_items ADD CONSTRAINT sale_return_items_product_id_fkey
            FOREIGN KEY (product_id) REFERENCES public.products(id) ON DELETE RESTRICT;
    END IF;
END;
$$;

CREATE INDEX IF NOT EXISTS idx_sale_return_items_return ON public.sale_return_items (sale_return_id);
CREATE INDEX IF NOT EXISTS idx_sale_return_items_product ON public.sale_return_items (product_id);
CREATE INDEX IF NOT EXISTS idx_sale_return_items_sale_item ON public.sale_return_items (sale_item_id);
CREATE INDEX IF NOT EXISTS idx_sale_return_items_outcome ON public.sale_return_items (outcome);

-- ---------------------------------------------------------------------------
-- sale_return_replacements
-- ---------------------------------------------------------------------------
-- A replacement unit physically delivered in exchange for the returned unit.
-- Linked to the return item it replaces; replacement quantity per line is
-- capped at the returned quantity (PRD §12.2 #13).

CREATE TABLE IF NOT EXISTS public.sale_return_replacements (
    id integer NOT NULL,
    sale_return_id integer NOT NULL,
    sale_return_item_id integer NOT NULL,
    product_id integer NOT NULL,
    quantity integer NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT sale_return_replacements_pkey PRIMARY KEY (id),
    CONSTRAINT chk_sale_return_replacement_qty CHECK ((quantity > 0))
);

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conrelid = 'public.sale_return_replacements'::regclass AND conname = 'sale_return_replacements_sale_return_id_fkey'
    ) THEN
        ALTER TABLE ONLY public.sale_return_replacements ADD CONSTRAINT sale_return_replacements_sale_return_id_fkey
            FOREIGN KEY (sale_return_id) REFERENCES public.sale_returns(id) ON DELETE CASCADE;
    END IF;
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conrelid = 'public.sale_return_replacements'::regclass AND conname = 'sale_return_replacements_sale_return_item_id_fkey'
    ) THEN
        ALTER TABLE ONLY public.sale_return_replacements ADD CONSTRAINT sale_return_replacements_sale_return_item_id_fkey
            FOREIGN KEY (sale_return_item_id) REFERENCES public.sale_return_items(id) ON DELETE CASCADE;
    END IF;
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conrelid = 'public.sale_return_replacements'::regclass AND conname = 'sale_return_replacements_product_id_fkey'
    ) THEN
        ALTER TABLE ONLY public.sale_return_replacements ADD CONSTRAINT sale_return_replacements_product_id_fkey
            FOREIGN KEY (product_id) REFERENCES public.products(id) ON DELETE RESTRICT;
    END IF;
END;
$$;

CREATE INDEX IF NOT EXISTS idx_sale_return_replacements_return ON public.sale_return_replacements (sale_return_id);
CREATE INDEX IF NOT EXISTS idx_sale_return_replacements_item ON public.sale_return_replacements (sale_return_item_id);

-- ---------------------------------------------------------------------------
-- sale_return_payments
-- ---------------------------------------------------------------------------
-- How the refund was paid out: cash from the drawer and/or non-cash back to
-- the original method (PRD §6 flow step 7). Rows are written only for
-- 'refund' lines — a replacement moves goods, not money.

CREATE TABLE IF NOT EXISTS public.sale_return_payments (
    id integer NOT NULL,
    sale_return_id integer NOT NULL,
    payment_method_id integer NOT NULL,
    payment_method_code character varying(50) NOT NULL,
    amount integer NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT sale_return_payments_pkey PRIMARY KEY (id),
    CONSTRAINT chk_sale_return_payment_amount CHECK ((amount > 0))
);

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conrelid = 'public.sale_return_payments'::regclass AND conname = 'sale_return_payments_sale_return_id_fkey'
    ) THEN
        ALTER TABLE ONLY public.sale_return_payments ADD CONSTRAINT sale_return_payments_sale_return_id_fkey
            FOREIGN KEY (sale_return_id) REFERENCES public.sale_returns(id) ON DELETE CASCADE;
    END IF;
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conrelid = 'public.sale_return_payments'::regclass AND conname = 'sale_return_payments_payment_method_id_fkey'
    ) THEN
        ALTER TABLE ONLY public.sale_return_payments ADD CONSTRAINT sale_return_payments_payment_method_id_fkey
            FOREIGN KEY (payment_method_id) REFERENCES public.payment_methods(id) ON DELETE RESTRICT;
    END IF;
END;
$$;

CREATE INDEX IF NOT EXISTS idx_sale_return_payments_return ON public.sale_return_payments (sale_return_id);

-- ---------------------------------------------------------------------------
-- return_approvals
-- ---------------------------------------------------------------------------
-- Created when the refund amount exceeds the app_settings
-- 'return.approval_threshold' (default 500000 IDR), or lazily for every
-- pending return. expires_at is the approval window
-- 'return.approval_timeout_minutes' (default 5).

CREATE TABLE IF NOT EXISTS public.return_approvals (
    id integer NOT NULL,
    sale_return_id integer NOT NULL,
    requested_by integer NOT NULL,
    approved_by integer,
    status character varying(20) DEFAULT 'pending' NOT NULL,
    reason text,
    rejected_reason text,
    expires_at timestamp with time zone NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT return_approvals_pkey PRIMARY KEY (id),
    CONSTRAINT chk_return_approval_status CHECK (((status)::text = ANY (ARRAY[('pending'::character varying)::text, ('approved'::character varying)::text, ('rejected'::character varying)::text, ('expired'::character varying)::text])))
);

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conrelid = 'public.return_approvals'::regclass AND conname = 'return_approvals_sale_return_id_fkey'
    ) THEN
        ALTER TABLE ONLY public.return_approvals ADD CONSTRAINT return_approvals_sale_return_id_fkey
            FOREIGN KEY (sale_return_id) REFERENCES public.sale_returns(id) ON DELETE CASCADE;
    END IF;
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conrelid = 'public.return_approvals'::regclass AND conname = 'return_approvals_requested_by_fkey'
    ) THEN
        ALTER TABLE ONLY public.return_approvals ADD CONSTRAINT return_approvals_requested_by_fkey
            FOREIGN KEY (requested_by) REFERENCES public.users(id) ON DELETE RESTRICT;
    END IF;
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conrelid = 'public.return_approvals'::regclass AND conname = 'return_approvals_approved_by_fkey'
    ) THEN
        ALTER TABLE ONLY public.return_approvals ADD CONSTRAINT return_approvals_approved_by_fkey
            FOREIGN KEY (approved_by) REFERENCES public.users(id) ON DELETE SET NULL;
    END IF;
END;
$$;

CREATE INDEX IF NOT EXISTS idx_return_approvals_return ON public.return_approvals (sale_return_id);
CREATE INDEX IF NOT EXISTS idx_return_approvals_status ON public.return_approvals (status, expires_at);

-- ---------------------------------------------------------------------------
-- Sequence ownership: id defaults
-- ---------------------------------------------------------------------------

-- Forward-only setval (matches 000_baseline.sql's sequence-repair loop): read
-- the sequence's current position and only bump it when MAX(id) has actually
-- advanced past it. The previous unconditional setval(seq, MAX(id)+1, false)
-- would REWIND the sequence whenever it had already moved ahead of MAX(id) —
-- e.g. ids consumed by rolled-back transactions or since-deleted rows — and
-- hand out sequence values a concurrent in-flight transaction had already
-- claimed, causing a primary-key collision. These sequences are all INCREMENT
-- BY 1, so the next value is last_value + (1 if is_called else 0).
DO $$
DECLARE
    seq       text;
    tbl       text;
    max_id    bigint;
    last_val  bigint;
    called    boolean;
    next_val  bigint;
    i         int;
    -- sequence -> owning table pairs (each table's id defaults to its sequence).
    pairs text[][] := ARRAY[
        ['public.sale_returns_id_seq',              'public.sale_returns'],
        ['public.sale_return_items_id_seq',         'public.sale_return_items'],
        ['public.sale_return_replacements_id_seq',  'public.sale_return_replacements'],
        ['public.sale_return_payments_id_seq',      'public.sale_return_payments'],
        ['public.return_approvals_id_seq',          'public.return_approvals']
    ];
BEGIN
    FOR i IN 1 .. array_length(pairs, 1) LOOP
        seq := pairs[i][1];
        tbl := pairs[i][2];
        EXECUTE format('SELECT COALESCE(MAX(id), 0) FROM %s', tbl) INTO max_id;
        EXECUTE format('SELECT last_value, is_called FROM %s', seq) INTO last_val, called;
        next_val := last_val + CASE WHEN called THEN 1 ELSE 0 END;
        IF max_id >= next_val THEN
            PERFORM setval(seq::regclass, max_id, true);
        END IF;
    END LOOP;
END;
$$;

ALTER TABLE public.sale_returns ALTER COLUMN id SET DEFAULT nextval('public.sale_returns_id_seq');
ALTER TABLE public.sale_return_items ALTER COLUMN id SET DEFAULT nextval('public.sale_return_items_id_seq');
ALTER TABLE public.sale_return_replacements ALTER COLUMN id SET DEFAULT nextval('public.sale_return_replacements_id_seq');
ALTER TABLE public.sale_return_payments ALTER COLUMN id SET DEFAULT nextval('public.sale_return_payments_id_seq');
ALTER TABLE public.return_approvals ALTER COLUMN id SET DEFAULT nextval('public.return_approvals_id_seq');
