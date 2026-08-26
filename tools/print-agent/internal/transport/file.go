package transport

import (
	"os"
	"path/filepath"
	"strings"
)

// FileTransport writes ESC/POS bytes to a .bin file. This is the development and
// CI transport: the generated file is the canonical renderer output and can be
// inspected or fed to a printer later.
type FileTransport struct {
	dir string
}

// NewFile creates a file transport writing into dir (defaults to the OS temp dir).
func NewFile(dir string) (*FileTransport, error) {
	if dir == "" {
		dir = os.TempDir()
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}
	return &FileTransport{dir: dir}, nil
}

// PathFor returns the absolute path that Write would use for a given job.
func (f *FileTransport) PathFor(jobID string) string {
	return filepath.Join(f.dir, "receipt-"+sanitize(jobID)+".bin")
}

// Write renders the receipt to a .bin file named after the job.
func (f *FileTransport) Write(jobID string, data []byte) error {
	return os.WriteFile(f.PathFor(jobID), data, 0o644)
}

// Close is a no-op for the file transport.
func (f *FileTransport) Close() error { return nil }

// Type reports the transport kind.
func (f *FileTransport) Type() string { return "file" }

func sanitize(s string) string {
	return strings.Map(func(r rune) rune {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '-', r == '_':
			return r
		default:
			return '_'
		}
	}, s)
}
