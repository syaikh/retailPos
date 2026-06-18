package repository

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestGenerateSlug_Basic(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"simple", "Hello World", "hello-world"},
		{"leading/trailing spaces", "  Hello World  ", "hello-world"},
		{"uppercase", "UPPERCASE", "uppercase"},
		{"apostrophe", "Cat's & Dog's", "cats-and-dogs"},
		{"consecutive hyphens", "foo---bar", "foo-bar"},
		{"trim hyphens", "-hello-", "hello"},
		{"ampersand", "Rock & Roll", "rock-and-roll"},
		{"slash", "red/blue/green", "red-blue-green"},
		{"plus", "C++", "cplusplus"},
		{"equals", "A = B", "a-equals-b"},
		{"question mark", "What?", "what"},
		{"exclamation", "Hello!", "hello"},
		{"at sign", "user@example", "useratexample"},
		{"hash", "Topic #1", "topic-number1"},
		{"percent", "100%", "100percent"},
		{"parentheses", "Foo (Bar)", "foo-bar"},
		{"empty string", "", ""},
		{"already slug", "hello-world", "hello-world"},
		{"mixed special chars", "@ Hello # World ?", "at-hello-number-world"},
		{"multiple special chars", `"test" & 'example'`, "test-and-example"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := generateSlug(tc.input)
			assert.Equal(t, tc.expected, got)
		})
	}
}

func TestGenerateSlug_Truncation(t *testing.T) {
	input := "a"
	for i := 0; i < 150; i++ {
		input += "b"
	}
	slug := generateSlug(input)
	assert.Equal(t, 120, len(slug), "slug should be truncated to 120 characters")
	assert.Equal(t, "a"+strings.Repeat("b", 119), slug)
}

func TestMustLoadJakarta(t *testing.T) {
	loc := mustLoadJakarta()
	assert.NotNil(t, loc, "location should not be nil")

	name := loc.String()
	assert.Equal(t, "Asia/Jakarta", name, "should be Asia/Jakarta")

	_, offset := time.Now().In(loc).Zone()
	assert.Equal(t, 7*3600, offset, "should have +07:00 offset")

	// Verify it returns the same instance on multiple calls
	loc2 := mustLoadJakarta()
	assert.True(t, loc == loc2, "mustLoadJakarta should return the same instance")
}
