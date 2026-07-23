CREATE TABLE IF NOT EXISTS dead_letter_events (
    id          BIGSERIAL PRIMARY KEY,
    event_type  TEXT        NOT NULL,
    payload     JSONB,
    error       TEXT        NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_dead_letter_events_created_at ON dead_letter_events (created_at);
CREATE INDEX idx_dead_letter_events_event_type ON dead_letter_events (event_type);
