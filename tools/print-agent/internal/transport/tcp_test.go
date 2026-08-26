package transport

import (
	"net"
	"testing"
	"time"
)

func TestTCPTransportWrite(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()

	received := make(chan []byte, 1)
	go func() {
		conn, err := ln.Accept()
		if err != nil {
			return
		}
		defer conn.Close()
		buf := make([]byte, 64)
		n, _ := conn.Read(buf)
		received <- buf[:n]
	}()

	tr, err := NewTCP(ln.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	if err := tr.Write("job1", []byte{0x1b, '@', 'A'}); err != nil {
		t.Fatal(err)
	}

	select {
	case got := <-received:
		if got[0] != 0x1b || got[1] != '@' {
			t.Fatalf("unexpected bytes: %v", got)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for bytes")
	}

	if err := tr.Health(); err != nil {
		t.Fatalf("health check failed: %v", err)
	}
}

func TestSerialTransportRequiresDevice(t *testing.T) {
	if _, err := NewSerial(""); err == nil {
		t.Fatal("expected error for empty device")
	}
}

func TestFileTransportWrite(t *testing.T) {
	dir := t.TempDir()
	tr, err := NewFile(dir)
	if err != nil {
		t.Fatal(err)
	}
	if err := tr.Write("job9", []byte{0x1b, '@'}); err != nil {
		t.Fatal(err)
	}
	if tr.Type() != "file" {
		t.Fatalf("unexpected type %s", tr.Type())
	}
}
