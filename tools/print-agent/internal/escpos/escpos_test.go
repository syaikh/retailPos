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

func TestDrawerPulse(t *testing.T) {
	// ESC p n1 n2: the exact bytes a till's operator will one day be staring
	// at in a logic analyser, so pin and width must survive unaltered.
	if !bytes.Equal(DrawerPulse(0, 25), []byte{esc, 'p', 0, 25}) {
		t.Fatalf("DrawerPulse(0,25) = %v, want ESC p 0 25", DrawerPulse(0, 25))
	}
	if !bytes.Equal(DrawerPulse(5, 40), []byte{esc, 'p', 5, 40}) {
		t.Fatalf("DrawerPulse(5,40) = %v, want ESC p 5 40", DrawerPulse(5, 40))
	}
	// A zero width would send a pulse the printer may not act on at all, so it
	// must become a real, defensible default instead.
	if !bytes.Equal(DrawerPulse(0, 0), []byte{esc, 'p', 0, DefaultDrawerPulseMs}) {
		t.Fatalf("DrawerPulse(0,0) = %v, want ESC p 0 %d", DrawerPulse(0, 0), DefaultDrawerPulseMs)
	}
}
