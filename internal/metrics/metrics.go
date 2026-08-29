package metrics

import "sync/atomic"

// Counter is a simple in-process monotonically increasing counter used for
// lightweight observability where an external metrics system (Prometheus,
// OTel) is not wired up. It is safe for concurrent use.
type Counter struct {
	v atomic.Int64
}

// Inc increments the counter by one.
func (c *Counter) Inc() { c.v.Add(1) }

// Add increments the counter by n.
func (c *Counter) Add(n int64) { c.v.Add(n) }

// Value returns the current count.
func (c *Counter) Value() int64 { return c.v.Load() }

// AuditWriteFailures counts audit-log write operations that failed (e.g. DB
// errors). It lets operators detect silent audit-trail loss, which was
// previously invisible because callers discarded the returned error.
var AuditWriteFailures = &Counter{}

// Snapshot returns a flat map of all metrics for export (e.g. the /metrics
// endpoint). Add new counters here as observability needs grow.
func Snapshot() map[string]int64 {
	return map[string]int64{
		"audit_write_failures": AuditWriteFailures.Value(),
	}
}
