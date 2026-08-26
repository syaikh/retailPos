// Package printer provides printer health/status reporting on top of a transport.
package printer

import "print-agent/internal/transport"

// Manager reports printer connectivity and identity.
type Manager struct {
	t transport.Transport
}

// New creates a printer manager for the given transport.
func New(t transport.Transport) *Manager {
	return &Manager{t: t}
}

// Status returns whether the printer is connected and its transport kind.
func (m *Manager) Status() (connected bool, kind string) {
	kind = m.t.Type()
	if hc, ok := m.t.(transport.HealthChecker); ok {
		return hc.Health() == nil, kind
	}
	// Transports without a health check (e.g. file) are always considered ready.
	return true, kind
}
