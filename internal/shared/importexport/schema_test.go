package importexportshared

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIntPtr(t *testing.T) {
	tests := []struct {
		name  string
		input int
	}{
		{"positive", 42},
		{"zero", 0},
		{"negative", -10},
		{"max int", 1<<63 - 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IntPtr(tt.input)
			assert.NotNil(t, result)
			assert.Equal(t, tt.input, *result)
		})
	}
}

func TestIntPtr_PointerIdentity(t *testing.T) {
	p1 := IntPtr(5)
	p2 := IntPtr(5)
	assert.Equal(t, *p1, *p2)
	assert.NotSame(t, p1, p2)
}

func TestFloat64Ptr(t *testing.T) {
	tests := []struct {
		name  string
		input float64
	}{
		{"positive", 3.14},
		{"zero", 0.0},
		{"negative", -2.5},
		{"very small", 0.000001},
		{"very large", 1e18},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Float64Ptr(tt.input)
			assert.NotNil(t, result)
			assert.Equal(t, tt.input, *result)
		})
	}
}

func TestFloat64Ptr_PointerIdentity(t *testing.T) {
	p1 := Float64Ptr(3.14)
	p2 := Float64Ptr(3.14)
	assert.Equal(t, *p1, *p2)
	assert.NotSame(t, p1, p2)
}
