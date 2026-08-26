package transport

import (
	"fmt"
	"net"
	"time"
)

// TCPTransport sends ESC/POS bytes to a raw TCP thermal printer (e.g. port 9100).
type TCPTransport struct {
	addr string
}

// NewTCP builds a TCP transport for addr (host:port). It errors if no address
// is configured.
func NewTCP(addr string) (*TCPTransport, error) {
	if addr == "" {
		return nil, fmt.Errorf("tcp transport requires PRINT_TCP_ADDR")
	}
	return &TCPTransport{addr: addr}, nil
}

// Write opens a connection and sends the bytes.
func (t *TCPTransport) Write(jobID string, data []byte) error {
	conn, err := net.DialTimeout("tcp", t.addr, 3*time.Second)
	if err != nil {
		return err
	}
	defer conn.Close()
	_, err = conn.Write(data)
	return err
}

// Close is a no-op for the TCP transport (connections are per-write).
func (t *TCPTransport) Close() error { return nil }

// Type reports the transport kind.
func (t *TCPTransport) Type() string { return "tcp" }

// Health checks whether the printer accepts a TCP connection.
func (t *TCPTransport) Health() error {
	conn, err := net.DialTimeout("tcp", t.addr, 1*time.Second)
	if err != nil {
		return err
	}
	return conn.Close()
}
