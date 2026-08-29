package metrics

import "testing"

func TestCounter(t *testing.T) {
	c := &Counter{}
	if c.Value() != 0 {
		t.Fatalf("expected 0, got %d", c.Value())
	}
	c.Inc()
	c.Inc()
	if c.Value() != 2 {
		t.Fatalf("expected 2, got %d", c.Value())
	}
	c.Add(3)
	if c.Value() != 5 {
		t.Fatalf("expected 5, got %d", c.Value())
	}
}

func TestSnapshotIncludesAuditFailures(t *testing.T) {
	AuditWriteFailures.Inc()
	snap := Snapshot()
	v, ok := snap["audit_write_failures"]
	if !ok {
		t.Fatal("audit_write_failures missing from snapshot")
	}
	if v < 1 {
		t.Fatalf("expected audit_write_failures >= 1, got %d", v)
	}
}
