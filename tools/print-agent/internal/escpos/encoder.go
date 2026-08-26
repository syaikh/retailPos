// Package escpos provides low-level ESC/POS command builders for 58mm thermal
// receipt printers. It is dependency-free: all output is plain byte slices.
package escpos

// ESC/POS control characters.
const (
	esc = 0x1B
	gs  = 0x1D
)

// Init resets the printer to its default state.
func Init() []byte { return []byte{esc, '@'} }

// CodepageCP858 selects code page 858 (Latin-1 + Euro). This renders Indonesian
// "Rp" and common accented store-name/footer text correctly. Most 58mm printers
// honour it; ASCII bytes are identical across code pages anyway.
func CodepageCP858() []byte { return []byte{esc, 't', 0x13} }

// Align sets text alignment: 0 left, 1 center, 2 right.
func Align(mode uint8) []byte { return []byte{esc, 'a', mode} }

// Bold toggles bold mode.
func Bold(on bool) []byte {
	if on {
		return []byte{esc, 'E', 1}
	}
	return []byte{esc, 'E', 0}
}

// FontSize sets width/height scaling (0 = normal, up to 7 = 8x).
func FontSize(w, h uint8) []byte {
	if w > 7 {
		w = 7
	}
	if h > 7 {
		h = 7
	}
	return []byte{gs, '!', (w << 4) | h}
}

// LineFeed advances the paper by n lines.
func LineFeed(n int) []byte {
	if n < 1 {
		n = 1
	}
	return []byte{esc, 'd', byte(n)}
}

// Cut performs a full paper cut.
func Cut() []byte { return []byte{gs, 'V', 0x00} }

// Text encodes s as CP858-compatible bytes and appends a newline.
func Text(s string) []byte { return append(encodeCP858(s), '\n') }

// TextRaw encodes s without a trailing newline.
func TextRaw(s string) []byte { return encodeCP858(s) }
