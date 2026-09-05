-- 041_shift_settings.sql
-- Configurable shift settings.

INSERT INTO app_settings (key, value) VALUES
    ('shift_discrepancy_threshold', '50000'),
    ('shift_blind_close', 'false')
ON CONFLICT (key) DO NOTHING;
