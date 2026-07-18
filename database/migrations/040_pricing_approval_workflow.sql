-- Migration 040: Pricing Approval Workflow
-- Add status column for draft/pending/approved/rejected workflow.
-- All existing rules default to 'approved' (grandfathered).

BEGIN;

ALTER TABLE pricing_rules
  ADD COLUMN status VARCHAR(20) NOT NULL DEFAULT 'approved';

ALTER TABLE pricing_rules
  ADD CONSTRAINT pricing_rules_status_check
  CHECK (status IN ('draft', 'pending', 'approved', 'rejected'));

CREATE INDEX idx_pricing_rules_status ON pricing_rules(status);

-- Set all existing active rules to 'approved'
UPDATE pricing_rules SET status = 'approved' WHERE is_active = true;

-- Set inactive rules to 'draft'
UPDATE pricing_rules SET status = 'draft' WHERE is_active = false;

COMMIT;
