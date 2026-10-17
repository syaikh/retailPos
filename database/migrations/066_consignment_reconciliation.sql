-- 066_consignment_reconciliation.sql
-- Phase 3 (PRD §15.10): physical reconciliation of consignment-owned stock.
--
-- consignment_stock.available_qty can drift from what is physically on the
-- shelf (damage, miscount, theft, unrecorded receipt). This adds a durable
-- per-arrangement reconciliation record (draft -> counted -> posted) and the
-- machinery to fold a store-fault shortfall into the supplier's next
-- settlement (D3b). Design: docs/design/consignment-reconciliation-spike.md.
--
-- Permanently re-runnable per the migration replay contract: every statement is
-- guarded (IF NOT EXISTS / guarded DO blocks). The permission row and grants
-- are mirrored in 000_baseline.sql (fresh-install parity, per 065).
--
-- The permission id (97) only matters here as the code; the baseline seeds it
-- by id. Both are kept in step by internal/permissions/baseline_test.go.

-- ---------------------------------------------------------------------------
-- Permission consignment.reconcile + grants (superadmin, manager, inventory_staff)
-- ---------------------------------------------------------------------------

DO $$
BEGIN
  INSERT INTO permissions (code, name, description)
  SELECT 'consignment.reconcile', 'Rekonsiliasi Stok Konsinyasi',
         'Bisa menghitung dan memposting rekonsiliasi fisik stok konsinyasi'
  WHERE NOT EXISTS (SELECT 1 FROM permissions WHERE code = 'consignment.reconcile');

  -- Counting stock is an inventory-staff job; approving the correction is a
  -- store-level decision, so manager and superadmin hold it too.
  INSERT INTO role_permissions (role_id, permission_id)
  SELECT r.id, p.id
  FROM roles r
  JOIN permissions p ON p.code = 'consignment.reconcile'
  WHERE r.name IN ('superadmin', 'manager', 'inventory_staff')
    AND NOT EXISTS (
      SELECT 1 FROM role_permissions rp
      WHERE rp.role_id = r.id AND rp.permission_id = p.id
    );

  -- inventory_staff could not previously see consignment records at all; the
  -- reconcile screen lists arrangements, so grant the read alongside it.
  INSERT INTO role_permissions (role_id, permission_id)
  SELECT r.id, p.id
  FROM roles r
  JOIN permissions p ON p.code = 'consignment.view'
  WHERE r.name = 'inventory_staff'
    AND NOT EXISTS (
      SELECT 1 FROM role_permissions rp
      WHERE rp.role_id = r.id AND rp.permission_id = p.id
    );
END $$;

-- ---------------------------------------------------------------------------
-- consignment_reconciliations (header)
-- ---------------------------------------------------------------------------

CREATE SEQUENCE IF NOT EXISTS public.consignment_reconciliation_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

CREATE TABLE IF NOT EXISTS public.consignment_reconciliations (
    id integer NOT NULL,
    reconciliation_number character varying(30) NOT NULL,
    arrangement_id integer NOT NULL,
    supplier_id integer NOT NULL,
    store_id integer NOT NULL,
    status character varying(20) DEFAULT 'draft'::character varying NOT NULL,
    notes text,
    counted_by integer,
    posted_by integer,
    created_by integer NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    posted_at timestamp with time zone,
    CONSTRAINT consignment_reconciliations_pkey PRIMARY KEY (id),
    CONSTRAINT consignment_reconciliations_number_key UNIQUE (reconciliation_number),
    CONSTRAINT chk_consignment_reconciliation_status CHECK (((status)::text = ANY (ARRAY[('draft'::character varying)::text, ('counted'::character varying)::text, ('posted'::character varying)::text])))
);

CREATE SEQUENCE IF NOT EXISTS public.consignment_reconciliations_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

ALTER SEQUENCE public.consignment_reconciliations_id_seq OWNED BY public.consignment_reconciliations.id;

ALTER TABLE ONLY public.consignment_reconciliations
    ALTER COLUMN id SET DEFAULT nextval('public.consignment_reconciliations_id_seq'::regclass);

-- ---------------------------------------------------------------------------
-- consignment_reconciliation_items
-- ---------------------------------------------------------------------------

CREATE TABLE IF NOT EXISTS public.consignment_reconciliation_items (
    id integer NOT NULL,
    reconciliation_id integer NOT NULL,
    product_id integer NOT NULL,
    system_available integer NOT NULL DEFAULT 0,
    physical_available integer NOT NULL DEFAULT 0,
    variance integer NOT NULL DEFAULT 0,
    reason character varying(30),
    liability_amount integer NOT NULL DEFAULT 0,
    settled_settlement_id integer,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT consignment_reconciliation_items_pkey PRIMARY KEY (id),
    CONSTRAINT chk_consignment_reconciliation_item_physical CHECK ((physical_available >= 0)),
    CONSTRAINT chk_consignment_reconciliation_item_liability CHECK ((liability_amount >= 0)),
    -- A non-zero variance is a correction, and a correction must say why.
    CONSTRAINT chk_consignment_reconciliation_item_reason CHECK (((variance = 0) OR (reason IS NOT NULL)))
);

CREATE SEQUENCE IF NOT EXISTS public.consignment_reconciliation_items_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

ALTER SEQUENCE public.consignment_reconciliation_items_id_seq OWNED BY public.consignment_reconciliation_items.id;

ALTER TABLE ONLY public.consignment_reconciliation_items
    ALTER COLUMN id SET DEFAULT nextval('public.consignment_reconciliation_items_id_seq'::regclass);

-- ---------------------------------------------------------------------------
-- consignment_settlements.total_liability (D3b)
-- ---------------------------------------------------------------------------

ALTER TABLE public.consignment_settlements
    ADD COLUMN IF NOT EXISTS total_liability integer NOT NULL DEFAULT 0;

-- ---------------------------------------------------------------------------
-- Foreign keys (guarded so a second runner pass is a no-op)
-- ---------------------------------------------------------------------------

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conrelid = 'public.consignment_reconciliations'::regclass
          AND conname = 'consignment_reconciliations_arrangement_id_fkey'
    ) THEN
        ALTER TABLE ONLY public.consignment_reconciliations
            ADD CONSTRAINT consignment_reconciliations_arrangement_id_fkey
            FOREIGN KEY (arrangement_id) REFERENCES public.consignment_arrangements(id);
    END IF;

    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conrelid = 'public.consignment_reconciliations'::regclass
          AND conname = 'consignment_reconciliations_supplier_id_fkey'
    ) THEN
        ALTER TABLE ONLY public.consignment_reconciliations
            ADD CONSTRAINT consignment_reconciliations_supplier_id_fkey
            FOREIGN KEY (supplier_id) REFERENCES public.suppliers(id);
    END IF;

    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conrelid = 'public.consignment_reconciliations'::regclass
          AND conname = 'consignment_reconciliations_store_id_fkey'
    ) THEN
        ALTER TABLE ONLY public.consignment_reconciliations
            ADD CONSTRAINT consignment_reconciliations_store_id_fkey
            FOREIGN KEY (store_id) REFERENCES public.stores(id);
    END IF;

    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conrelid = 'public.consignment_reconciliations'::regclass
          AND conname = 'consignment_reconciliations_created_by_fkey'
    ) THEN
        ALTER TABLE ONLY public.consignment_reconciliations
            ADD CONSTRAINT consignment_reconciliations_created_by_fkey
            FOREIGN KEY (created_by) REFERENCES public.users(id);
    END IF;

    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conrelid = 'public.consignment_reconciliations'::regclass
          AND conname = 'consignment_reconciliations_counted_by_fkey'
    ) THEN
        ALTER TABLE ONLY public.consignment_reconciliations
            ADD CONSTRAINT consignment_reconciliations_counted_by_fkey
            FOREIGN KEY (counted_by) REFERENCES public.users(id) ON DELETE SET NULL;
    END IF;

    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conrelid = 'public.consignment_reconciliations'::regclass
          AND conname = 'consignment_reconciliations_posted_by_fkey'
    ) THEN
        ALTER TABLE ONLY public.consignment_reconciliations
            ADD CONSTRAINT consignment_reconciliations_posted_by_fkey
            FOREIGN KEY (posted_by) REFERENCES public.users(id) ON DELETE SET NULL;
    END IF;

    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conrelid = 'public.consignment_reconciliation_items'::regclass
          AND conname = 'consignment_reconciliation_items_reconciliation_id_fkey'
    ) THEN
        ALTER TABLE ONLY public.consignment_reconciliation_items
            ADD CONSTRAINT consignment_reconciliation_items_reconciliation_id_fkey
            FOREIGN KEY (reconciliation_id) REFERENCES public.consignment_reconciliations(id) ON DELETE CASCADE;
    END IF;

    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conrelid = 'public.consignment_reconciliation_items'::regclass
          AND conname = 'consignment_reconciliation_items_product_id_fkey'
    ) THEN
        ALTER TABLE ONLY public.consignment_reconciliation_items
            ADD CONSTRAINT consignment_reconciliation_items_product_id_fkey
            FOREIGN KEY (product_id) REFERENCES public.products(id) ON DELETE CASCADE;
    END IF;

    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conrelid = 'public.consignment_reconciliation_items'::regclass
          AND conname = 'consignment_reconciliation_items_settled_settlement_id_fkey'
    ) THEN
        ALTER TABLE ONLY public.consignment_reconciliation_items
            ADD CONSTRAINT consignment_reconciliation_items_settled_settlement_id_fkey
            FOREIGN KEY (settled_settlement_id) REFERENCES public.consignment_settlements(id) ON DELETE SET NULL;
    END IF;
END $$;

-- ---------------------------------------------------------------------------
-- Indexes
-- ---------------------------------------------------------------------------

CREATE INDEX IF NOT EXISTS idx_consignment_reconciliations_scope
    ON public.consignment_reconciliations (supplier_id, store_id, status);

CREATE INDEX IF NOT EXISTS idx_consignment_reconciliation_items_reconciliation
    ON public.consignment_reconciliation_items (reconciliation_id);

-- Partial index for the settlement pull: open (unsettled) liabilities only.
CREATE INDEX IF NOT EXISTS idx_consignment_reconciliation_items_open_liability
    ON public.consignment_reconciliation_items (settled_settlement_id)
    WHERE settled_settlement_id IS NULL;
