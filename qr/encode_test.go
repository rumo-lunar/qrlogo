package qr_test

import (
	"strings"
	"testing"

	"github.com/rumo-lunar/qrlogo/qr"
)

func TestEncodeData_CodewordCountIs2334(t *testing.T) {
	// Arrange + Act
	cw, _ := qr.EncodeData("https://lunar.app")

	// Assert
	if len(cw) != qr.DataCodewords {
		t.Fatalf("len(codewords) = %d, want %d", len(cw), qr.DataCodewords)
	}
}

func TestEncodeData_DomainHasFreeVars(t *testing.T) {
	// Arrange + Act
	_, d := qr.EncodeData("https://lunar.app")

	// Assert
	if d.NumVars <= 0 {
		t.Fatalf("NumVars = %d, want > 0", d.NumVars)
	}
}

func TestEncodeData_DomainSizeShrinksWithURLLength(t *testing.T) {
	cases := []struct {
		urlLen      int
		wantNumVars int
	}{
		{1, (qr.DataCodewords - 4) * 8},
		{100, (qr.DataCodewords - 103) * 8},
		{qr.MaxURLBytes, (qr.DataCodewords - qr.MaxURLBytes - 3) * 8},
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

func TestEncodeData_MaxURLDoesNotPanic(t *testing.T) {
	// Arrange + Act + Assert
	url := strings.Repeat("x", qr.MaxURLBytes)
	cw, d := qr.EncodeData(url)

	if len(cw) != qr.DataCodewords {
		t.Errorf("codewords length = %d, want %d", len(cw), qr.DataCodewords)
	}
	// MaxURLBytes bytes → (DataCodewords - MaxURLBytes - 3) * 8 free vars = 0.
	wantVars := (qr.DataCodewords - qr.MaxURLBytes - 3) * 8
	if d.NumVars != wantVars {
		t.Errorf("NumVars = %d, want %d", d.NumVars, wantVars)
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
	qr.EncodeData(strings.Repeat("x", qr.MaxURLBytes+1))
}
