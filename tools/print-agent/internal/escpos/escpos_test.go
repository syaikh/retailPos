package escpos

import (
	"bytes"
	"testing"
)

func TestInitAndCut(t *testing.T) {
	out := NewBuilder().Write(Init()).Write(Cut()).Bytes()
	if !bytes.Contains(out, []byte{esc, '@'}) {
		t.Fatal("missing ESC @ init")
	}
	if !bytes.Contains(out, []byte{gs, 'V', 0x00}) {
		t.Fatal("missing GS V cut")
	}
}

func TestEncodeCP858Fallback(t *testing.T) {
	got := encodeCP858("Rp 10.000 Café")
	want := "Rp 10.000 Cafe"
	if string(got) != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestAlignAndBold(t *testing.T) {
	if !bytes.Equal(Align(1), []byte{esc, 'a', 1}) {
		t.Fatal("Align(center) wrong")
	}
	if !bytes.Equal(Bold(true), []byte{esc, 'E', 1}) {
		t.Fatal("Bold(true) wrong")
	}
}
