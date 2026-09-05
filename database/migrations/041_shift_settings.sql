-- 041_shift_settings.sql
-- Configurable shift settings.

INSERT INTO app_settings (key, value) VALUES
    ('shift_discrepancy_threshold', '50000'),
    ('shift_blind_close', 'false'),
    ('shift_auto_close_hours', '24')
ON CONFLICT (key) DO NOTHING;
