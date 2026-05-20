package qr_test

import (
	"strings"
	"testing"

	"github.com/rumo-lunar/qrlogo/qr"
)

func TestEncodeDataV40M_CodewordCountIs2334(t *testing.T) {
	// Arrange + Act
	cw, _ := qr.EncodeDataV40M("https://lunar.app")

	// Assert
	if len(cw) != qr.DataCodewordsV40M {
		t.Fatalf("len(codewords) = %d, want %d", len(cw), qr.DataCodewordsV40M)
	}
}

func TestEncodeDataV40M_NonNilDomain(t *testing.T) {
	// Arrange + Act
	_, d := qr.EncodeDataV40M("https://lunar.app")

	// Assert
	if d == nil {
		t.Fatal("domain is nil")
	}
	if d.NumVars <= 0 {
		t.Errorf("NumVars = %d, want > 0", d.NumVars)
	}
}

func TestEncodeDataV40M_DomainSizeShrinksWithURLLength(t *testing.T) {
	cases := []struct {
		urlLen      int
		wantNumVars int
	}{
		{1, (qr.DataCodewordsV40M - 4) * 8},
		{100, (qr.DataCodewordsV40M - 103) * 8},
		{qr.MaxURLBytesV40M, (qr.DataCodewordsV40M - qr.MaxURLBytesV40M - 3) * 8},
	}
	for _, c := range cases {
		// Arrange
		url := strings.Repeat("a", c.urlLen)

		// Act
		_, d := qr.EncodeDataV40M(url)

		// Assert
		if d.NumVars != c.wantNumVars {
			t.Errorf("urlLen=%d: NumVars=%d, want %d", c.urlLen, d.NumVars, c.wantNumVars)
		}
	}
}

func TestEncodeDataV40M_MaxURLDoesNotPanic(t *testing.T) {
	// Arrange + Act + Assert
	url := strings.Repeat("x", qr.MaxURLBytesV40M)
	cw, d := qr.EncodeDataV40M(url)

	if len(cw) != qr.DataCodewordsV40M {
		t.Errorf("codewords length = %d, want %d", len(cw), qr.DataCodewordsV40M)
	}
	// MaxURLBytesV40M bytes → (DataCodewordsV40M - MaxURLBytesV40M - 3) * 8 free vars = 0.
	wantVars := (qr.DataCodewordsV40M - qr.MaxURLBytesV40M - 3) * 8
	if d.NumVars != wantVars {
		t.Errorf("NumVars = %d, want %d", d.NumVars, wantVars)
	}
}

func TestEncodeDataV40M_Empty_Panics(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("expected panic on empty URL")
		}
	}()
	qr.EncodeDataV40M("")
}

func TestEncodeDataV40M_TooLong_Panics(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("expected panic on over-length URL")
		}
	}()
	qr.EncodeDataV40M(strings.Repeat("x", qr.MaxURLBytesV40M+1))
}
