package shared

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLogger_ReturnsNonNil(t *testing.T) {
	logger := Logger()
	require.NotNil(t, logger)
}

func TestLogger_SingletonInstance(t *testing.T) {
	l1 := Logger()
	l2 := Logger()
	assert.Same(t, l1, l2)
}

func TestLogger_Functional(t *testing.T) {
	logger := Logger()
	// Should not panic when logging
	logger.Info("test message", "key", "value")
	logger.Debug("debug message", "key", 42)
	logger.Warn("warn message")
}
