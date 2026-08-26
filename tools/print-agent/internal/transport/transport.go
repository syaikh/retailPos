// Package transport defines how rendered ESC/POS bytes reach a printer.
package transport

// Transport writes rendered receipt bytes to a physical or virtual printer.
// The jobID is used by file/serial transports for naming; network transports
// ignore it.
type Transport interface {
	Write(jobID string, data []byte) error
	Close() error
	Type() string
}

// HealthChecker is implemented by transports that can report connectivity.
type HealthChecker interface {
	Health() error
}

// Config selects and configures a transport.
type Config struct {
	Kind         string // file | tcp | serial
	OutputDir    string // file transport output directory
	TCPAddr      string // tcp transport host:port
	SerialDevice string // serial/usb transport device path
}

// New builds a transport from the configuration.
func New(cfg Config) (Transport, error) {
	switch cfg.Kind {
	case "tcp":
		return NewTCP(cfg.TCPAddr)
	case "serial", "usb":
		return NewSerial(cfg.SerialDevice)
	case "file", "":
		return NewFile(cfg.OutputDir)
	default:
		return nil, &TransportError{Kind: cfg.Kind}
	}
}

// TransportError indicates an unknown or misconfigured transport kind.
type TransportError struct {
	Kind string
}

func (e *TransportError) Error() string {
	return "unknown or unsupported transport: " + e.Kind
}
