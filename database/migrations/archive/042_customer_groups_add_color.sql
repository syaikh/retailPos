-- Migration 042: Add color column to customer_groups
-- Supports visual differentiation for customer group avatars

ALTER TABLE customer_groups ADD COLUMN IF NOT EXISTS color VARCHAR(7);

-- Update seed data with default colors
UPDATE customer_groups SET color = '#6C5CE7' WHERE name = 'VIP';
UPDATE customer_groups SET color = '#00B894' WHERE name = 'Member';
UPDATE customer_groups SET color = '#636E72' WHERE name = 'Walk-in';

-- Set default color for any groups without a color
UPDATE customer_groups SET color = '#6C5CE7' WHERE color IS NULL;
