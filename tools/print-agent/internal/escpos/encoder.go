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

// DefaultDrawerPulseMs is the pulse length used when a caller passes 0.
const DefaultDrawerPulseMs uint8 = 25

// DrawerPulse generates the electrical pulse that opens a cash drawer wired to
// the printer's drawer port (RJ11 on almost every printer sold for this
// purpose). The drawer is powered and driven by the printer, not by the PC, so
// this single command is the whole of cash-drawer support: nothing on the host
// side ever sees the drawer.
//
// n1 selects the connector pin the pulse is applied to. 0 drives pin 2 and 1
// drives pin 5 per the EPSON reference; 2 and 5 are accepted because ESC/POS
// compatible printers use them as synonyms for the same two pins. Any other
// value is left as-is rather than guessed at: an unsupported pin opens nothing,
// and silently rewriting it would hide the misconfiguration.
//
// n2 is the pulse length in milliseconds (1-255). Drawers only need a couple of
// milliseconds to latch, and 25ms is comfortably above that on a solenoid whose
// drive circuit is unknown. n2 of 0 is not a legal pulse length, so it falls
// back to DefaultDrawerPulseMs instead of sending a pulse the printer may
// interpret as no pulse at all.
func DrawerPulse(pin, onMs uint8) []byte {
	if onMs == 0 {
		onMs = DefaultDrawerPulseMs
	}
	return []byte{esc, 'p', pin, onMs}
}

// Text encodes s as CP858-compatible bytes and appends a newline.
func Text(s string) []byte { return append(encodeCP858(s), '\n') }

// TextRaw encodes s without a trailing newline.
func TextRaw(s string) []byte { return encodeCP858(s) }
