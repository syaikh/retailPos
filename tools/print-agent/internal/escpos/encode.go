package escpos

import "bytes"

// latin1Fallback maps a few common non-ASCII accented Latin letters and
// typographic characters to a best-fit ASCII representation. ESC/POS printers
// in CP858 mode render ASCII bytes identically, so this keeps store names and
// footers readable without pulling in a full code-page library.
var latin1Fallback = map[rune][]byte{
	'À': []byte("A"), 'Á': []byte("A"), 'Â': []byte("A"), 'Ã': []byte("A"), 'Ä': []byte("A"), 'Å': []byte("A"),
	'à': []byte("a"), 'á': []byte("a"), 'â': []byte("a"), 'ã': []byte("a"), 'ä': []byte("a"), 'å': []byte("a"),
	'È': []byte("E"), 'É': []byte("E"), 'Ê': []byte("E"), 'Ë': []byte("E"),
	'è': []byte("e"), 'é': []byte("e"), 'ê': []byte("e"), 'ë': []byte("e"),
	'Ì': []byte("I"), 'Í': []byte("I"), 'Î': []byte("I"), 'Ï': []byte("I"),
	'ì': []byte("i"), 'í': []byte("i"), 'î': []byte("i"), 'ï': []byte("i"),
	'Ò': []byte("O"), 'Ó': []byte("O"), 'Ô': []byte("O"), 'Õ': []byte("O"), 'Ö': []byte("O"),
	'ò': []byte("o"), 'ó': []byte("o"), 'ô': []byte("o"), 'õ': []byte("o"), 'ö': []byte("o"),
	'Ù': []byte("U"), 'Ú': []byte("U"), 'Û': []byte("U"), 'Ü': []byte("U"),
	'ù': []byte("u"), 'ú': []byte("u"), 'û': []byte("u"), 'ü': []byte("u"),
	'Ñ': []byte("N"), 'ñ': []byte("n"),
	'Ç': []byte("C"), 'ç': []byte("c"),
	'’': []byte("'"), '‘': []byte("'"), '“': []byte("\""), '”': []byte("\""), '–': []byte("-"), '—': []byte("-"),
	'&': []byte("&"),
}

// encodeCP858 converts a string to bytes for a CP858 printer: ASCII pass-through,
// best-effort accented-letter fallback, '?' for anything else unrepresentable.
func encodeCP858(s string) []byte {
	var buf bytes.Buffer
	for _, r := range s {
		if r < 0x80 {
			buf.WriteByte(byte(r))
			continue
		}
		if alt, ok := latin1Fallback[r]; ok {
			buf.Write(alt)
			continue
		}
		buf.WriteByte('?')
	}
	return buf.Bytes()
}
