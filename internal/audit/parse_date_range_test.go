package audit

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseDateRange_Empty(t *testing.T) {
	start, end, err := parseDateRange("", "")
	assert.NoError(t, err)
	assert.Nil(t, start)
	assert.Nil(t, end)
}

func TestParseDateRange_RFC3339(t *testing.T) {
	start, end, err := parseDateRange("2024-01-15T00:00:00Z", "2024-01-20T23:59:59Z")
	assert.NoError(t, err)
	assert.NotNil(t, start)
	assert.NotNil(t, end)
	assert.Equal(t, 2024, start.Year())
	assert.Equal(t, 1, int(start.Month()))
	assert.Equal(t, 15, start.Day())
	assert.Equal(t, 20, end.Day())
}

func TestParseDateRange_JakartaFormat(t *testing.T) {
	start, end, err := parseDateRange("2024-01-15", "2024-01-20")
	assert.NoError(t, err)
	assert.NotNil(t, start)
	assert.NotNil(t, end)
	assert.Equal(t, 2024, start.Year())
	assert.Equal(t, 1, int(start.Month()))
	assert.Equal(t, 15, start.Day())
	assert.Equal(t, 20, end.Day())
}

func TestParseDateRange_OnlyStart(t *testing.T) {
	start, end, err := parseDateRange("2024-01-15", "")
	assert.NoError(t, err)
	assert.NotNil(t, start)
	assert.Nil(t, end)
}

func TestParseDateRange_OnlyEnd(t *testing.T) {
	start, end, err := parseDateRange("", "2024-01-20")
	assert.NoError(t, err)
	assert.Nil(t, start)
	assert.NotNil(t, end)
}

func TestParseDateRange_InvalidStart(t *testing.T) {
	_, _, err := parseDateRange("not-a-date", "")
	assert.Error(t, err)
}

func TestParseDateRange_InvalidEnd(t *testing.T) {
	_, _, err := parseDateRange("", "not-a-date")
	assert.Error(t, err)
}

func TestParseDateRange_InvalidBoth(t *testing.T) {
	_, _, err := parseDateRange("bad-start", "bad-end")
	assert.Error(t, err)
}
