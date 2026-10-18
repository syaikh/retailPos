package escpos

import (
	"bytes"
	"testing"
	"unicode/utf8"
)

// FuzzEncodeCP858 exercises the receipt text encoder on arbitrary UTF-8 (and
// deliberately invalid) input. A CP858 printer in the configured mode expects
// single-byte character data, so every emitted byte must stay below 0x80: the
// encoder only ever writes ASCII pass-through, an ASCII best-fit fallback, or
// '?'. That contract is what keeps a customer name or a store footer from
// shifting the printer into an unintended code page mid-receipt.
func FuzzEncodeCP858(f *testing.F) {
	f.Add("")
	f.Add("Toko Sejahtera")
	f.Add("Café Ãçént")
	f.Add("50% OFF — limited")
	f.Add("日本語のテキスト")
	f.Add("\x00\x01\x7f")
	f.Add("\xff\xfe")

	f.Fuzz(func(t *testing.T, s string) {
		out := encodeCP858(s)

		for i, b := range out {
			if b >= 0x80 {
				t.Fatalf("encodeCP858(%q)[%d] = 0x%02x, want < 0x80", s, i, b)
			}
		}
		// Every rune maps to exactly one byte (pass-through, fallback, or '?'),
		// so a printer's character budget matches the rune count the receipt
		// layout was measured against.
		if want := utf8.RuneCountInString(s); len(out) != want {
			t.Fatalf("encodeCP858(%q) produced %d bytes for %d runes", s, len(out), want)
		}
		if asciiOnly(s) && string(out) != s {
			t.Fatalf("encodeCP858(%q) altered pure-ASCII input to %q", s, out)
		}
		if again := encodeCP858(s); !bytes.Equal(out, again) {
			t.Fatalf("encodeCP858(%q) not deterministic", s)
		}
	})
}

// asciiOnly reports whether every rune in s is plain ASCII.
func asciiOnly(s string) bool {
	for _, r := range s {
		if r >= 0x80 {
			return false
		}
	}
	return true
}
