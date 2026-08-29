-- P2 audit hardening (#9): add a request correlation ID to every audit row so
-- related events from a single HTTP request can be traced together.
ALTER TABLE audit_logs ADD COLUMN IF NOT EXISTS correlation_id TEXT;

COMMENT ON COLUMN audit_logs.correlation_id IS
	'Request correlation ID (X-Request-ID) for tracing related audit events within a single request.';
