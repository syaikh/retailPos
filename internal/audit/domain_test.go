package audit

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

type stubAuditCreator struct {
	err    error
	called bool
}

func (s *stubAuditCreator) CreateAuditLog(_ context.Context, _ *Log) error {
	s.called = true
	return s.err
}

func TestWriteFailClosed(t *testing.T) {
	t.Run("nil service is a no-op success", func(t *testing.T) {
		assert.True(t, WriteFailClosed(context.Background(), nil, &Log{Action: "x"}))
	})

	t.Run("successful write returns true", func(t *testing.T) {
		stub := &stubAuditCreator{}
		ok := WriteFailClosed(context.Background(), stub, &Log{Action: "x"})
		assert.True(t, ok)
		assert.True(t, stub.called)
	})

	t.Run("failed write returns false and signals abort", func(t *testing.T) {
		stub := &stubAuditCreator{err: errors.New("boom")}
		ok := WriteFailClosed(context.Background(), stub, &Log{Action: "x"})
		assert.False(t, ok)
		assert.True(t, stub.called)
	})
}
