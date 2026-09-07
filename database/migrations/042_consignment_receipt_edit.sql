-- Consignment receipt edit audit trail (append-only)
CREATE TABLE IF NOT EXISTS consignment_receipt_edits (
    id SERIAL,
    receipt_id INTEGER NOT NULL,
    edited_by INTEGER NOT NULL,
    edited_at TIMESTAMP WITH TIME ZONE DEFAULT NOW() NOT NULL,
    field TEXT NOT NULL,
    old_value TEXT NOT NULL,
    new_value TEXT NOT NULL,
    reason TEXT NOT NULL,
    ip_address INET,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW() NOT NULL,
    CONSTRAINT consignment_receipt_edits_pkey PRIMARY KEY (id),
    CONSTRAINT consignment_receipt_edits_receipt_id_fkey FOREIGN KEY (receipt_id)
        REFERENCES consignment_receipts(id) ON DELETE CASCADE,
    CONSTRAINT consignment_receipt_edits_edited_by_fkey FOREIGN KEY (edited_by)
        REFERENCES users(id) ON DELETE SET NULL
);

CREATE INDEX IF NOT EXISTS idx_consignment_receipt_edits_receipt
    ON consignment_receipt_edits (receipt_id);

CREATE INDEX IF NOT EXISTS idx_consignment_receipt_edits_edited_by
    ON consignment_receipt_edits (edited_by);

CREATE INDEX IF NOT EXISTS idx_consignment_receipt_edits_edited_at
    ON consignment_receipt_edits (edited_at);

-- Append-only trigger
CREATE OR REPLACE FUNCTION prevent_consignment_receipt_edit_modification()
RETURNS TRIGGER AS $$
BEGIN
    RAISE EXCEPTION 'consignment_receipt_edits is append-only: modifications are not permitted';
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trg_consignment_receipt_edits_immutable
    BEFORE UPDATE OR DELETE ON consignment_receipt_edits
    FOR EACH ROW
    EXECUTE FUNCTION prevent_consignment_receipt_edit_modification();

COMMENT ON TABLE consignment_receipt_edits IS 'Immutable audit trail for consignment receipt edits';
