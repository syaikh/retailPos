-- Add review fields to shifts for discrepancy threshold flagging

ALTER TABLE shifts ADD COLUMN IF NOT EXISTS needs_review BOOLEAN NOT NULL DEFAULT false;
ALTER TABLE shifts ADD COLUMN IF NOT EXISTS reviewed_by INTEGER REFERENCES users(id);
ALTER TABLE shifts ADD COLUMN IF NOT EXISTS reviewed_at TIMESTAMPTZ;
