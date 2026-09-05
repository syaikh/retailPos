-- 040_shift_cash_movements.sql
-- Cash movement tracking for shift reconciliation.

CREATE TABLE IF NOT EXISTS cash_movements (
    id              SERIAL PRIMARY KEY,
    shift_id        INTEGER NOT NULL REFERENCES shifts(id) ON DELETE RESTRICT,
    user_id         INTEGER NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    type            VARCHAR(20) NOT NULL,
    amount          INTEGER NOT NULL,
    description     TEXT,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT cash_movements_type_check
        CHECK (type IN ('cash_drop', 'paid_in', 'paid_out')),
    CONSTRAINT cash_movements_amount_check
        CHECK (amount > 0)
);

CREATE INDEX idx_cash_movements_shift_id ON cash_movements(shift_id);
CREATE INDEX idx_cash_movements_shift_type ON cash_movements(shift_id, type);

-- Permissions
INSERT INTO permissions (code, name, description) VALUES
    ('shift.cash_movement', 'Shift Cash Movement', 'Record cash drop / paid in / paid out');

-- Cashier: record own movements on own shifts
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM roles r, permissions p
WHERE r.name = 'cashier' AND p.code = 'shift.cash_movement';

-- Manager, admin, superadmin: full access
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM roles r, permissions p
WHERE r.name IN ('manager', 'admin', 'superadmin')
AND p.code = 'shift.cash_movement';
