package qr_test

import (
	"strings"
	"testing"

	"github.com/rumo-lunar/qrlogo/qr"
)

func TestEncodeData_CodewordCountIs254(t *testing.T) {
	// Arrange + Act
	cw, _ := qr.EncodeData("https://example.com")

	// Assert
	if len(cw) != qr.DataCodewordsV11M {
		t.Fatalf("len(codewords) = %d, want %d", len(cw), qr.DataCodewordsV11M)
	}
}

func TestEncodeData_DomainSizeShrinksWithURLLength(t *testing.T) {
	cases := []struct {
		urlLen      int
		wantNumVars int
	}{
		{1, (qr.DataCodewordsV11M - 4) * 8},                                   // 2000
		{50, (qr.DataCodewordsV11M - 53) * 8},                                 // 1608
		{100, (qr.DataCodewordsV11M - 103) * 8},                               // 1208
		{qr.MaxURLBytesV11M, (qr.DataCodewordsV11M - qr.MaxURLBytesV11M - 3) * 8}, // 1208
	}
	for _, c := range cases {
		// Arrange
		url := strings.Repeat("a", c.urlLen)

		// Act
		_, d := qr.EncodeData(url)

		// Assert
		if d.NumVars != c.wantNumVars {
			t.Errorf("urlLen=%d: NumVars=%d, want %d", c.urlLen, d.NumVars, c.wantNumVars)
		}
	}
}

func TestEncodeData_PrefixIsConstant_a(t *testing.T) {
	// Arrange: n=1, framing 4+16+8+4 = 32 bits = 4 codewords:
	//   01000000 00000000 00010110 00010000  →  0x40 0x00 0x16 0x10
	cw, d := qr.EncodeData("a")
	sol := make([]byte, (d.NumVars+7)/8)

	want := []byte{0x40, 0x00, 0x16, 0x10}

	// Act + Assert: the first 4 codewords must resolve to the
	// expected fixed bytes regardless of the solution (they hold no
	// variables).
	for i, w := range want {
		if got := d.ResolveByte(cw[i], sol); got != w {
			t.Errorf("codewords[%d] = 0x%02x, want 0x%02x", i, got, w)
		}
	}
}

func TestEncodeData_PrefixIsConstant_hi(t *testing.T) {
	// Arrange: n=2, framing 4+16+16+4 = 40 bits = 5 codewords:
	//   01000000 00000000 00100110 10000110 10010000
	//   →  0x40 0x00 0x26 0x86 0x90
	cw, d := qr.EncodeData("hi")
	sol := make([]byte, (d.NumVars+7)/8)

	want := []byte{0x40, 0x00, 0x26, 0x86, 0x90}

	// Act + Assert
	for i, w := range want {
		if got := d.ResolveByte(cw[i], sol); got != w {
			t.Errorf("codewords[%d] = 0x%02x, want 0x%02x", i, got, w)
		}
	}
}

func TestEncodeData_PaddingResolvesToSolutionBytes(t *testing.T) {
	// Arrange
	url := "abc" // n=3, 6 framing codewords, then 248 padding codewords
	cw, d := qr.EncodeData(url)

	// Build a solution whose bytes are 0..247 (mod 256), one byte per
	// padding codeword. By construction, padding codeword k holds
	// variables (8k)..(8k+7), and sym.ResolveByte reads variable n
	// from solution[n/8] MSB-first within the byte — so codeword
	// (n+3)+k resolves to solution[k].
	paddingCount := qr.DataCodewordsV11M - (len(url) + 3)
	sol := make([]byte, paddingCount)
	for k := 0; k < paddingCount; k++ {
		sol[k] = byte(k & 0xFF)
	}

	// Act + Assert
	for k := 0; k < paddingCount; k++ {
		got := d.ResolveByte(cw[len(url)+3+k], sol)
		want := sol[k]
		if got != want {
			t.Errorf("padding codeword %d resolved to 0x%02x, want 0x%02x",
				k, got, want)
		}
	}
}

func TestEncodeData_MaxLength_DoesNotPanic(t *testing.T) {
	// Arrange + Act + Assert
	url := strings.Repeat("x", qr.MaxURLBytesV11M)
	cw, d := qr.EncodeData(url)

	if len(cw) != qr.DataCodewordsV11M {
		t.Errorf("codewords length = %d, want %d", len(cw), qr.DataCodewordsV11M)
	}
	if d.NumVars != 1208 {
		t.Errorf("NumVars = %d, want 1208 (the V11-M / 100-char free-bit budget)", d.NumVars)
	}
}

func TestEncodeData_Empty_Panics(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("expected panic on empty URL")
		}
	}()
	qr.EncodeData("")
}

func TestEncodeData_TooLong_Panics(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("expected panic on over-length URL")
		}
	}()
	qr.EncodeData(strings.Repeat("x", qr.MaxURLBytesV11M+1))
}

func TestEncodeData_FirstCodewordEncodesModeAndLengthHighNibble(t *testing.T) {
	// Arrange: the first byte is always [mode=0100][len[15:12]]. For
	// n=100, len[15:12] = 0 (100 < 4096), so byte 0 = 0x40 for any
	// URL up to 4095 bytes.
	for _, urlLen := range []int{1, 7, 50, 100} {
		url := strings.Repeat("z", urlLen)
		cw, d := qr.EncodeData(url)
		sol := make([]byte, (d.NumVars+7)/8)

		// Act
		got := d.ResolveByte(cw[0], sol)

		// Assert
		if got != 0x40 {
			t.Errorf("urlLen=%d: codewords[0] = 0x%02x, want 0x40", urlLen, got)
		}
	}
}
