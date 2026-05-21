package qr_test

import (
	"testing"

	"github.com/rumo-lunar/qrlogo/qr"
	"github.com/rumo-lunar/qrlogo/qr/spec"
)

func TestBuild_GridSizeMatchesSpec(t *testing.T) {
	for _, v := range []spec.Version{1, 7, 10, 20, 40} {
		for _, ec := range []spec.ECLevel{spec.ECLow, spec.ECMedium, spec.ECQuartile, spec.ECHigh} {
			// Arrange
			s, _ := spec.New(v, ec)
			payload := []byte("https://lunar.app")
			if len(payload) > s.MaxByteModePayload() {
				continue // V1-H can't hold 17 bytes
			}

			// Act
			sym, err := qr.Build(payload, s)
			if err != nil {
				t.Fatalf("Build(%s): %v", s, err)
			}

			// Assert
			n := s.Version.Size()
			if len(sym.Grid) != n {
				t.Errorf("%s: rows = %d, want %d", s, len(sym.Grid), n)
			}
			for r, row := range sym.Grid {
				if len(row) != n {
					t.Errorf("%s: row %d width = %d, want %d", s, r, len(row), n)
				}
			}
		}
	}
}

func TestBuild_FinderPatternsAtThreeCorners(t *testing.T) {
	// Arrange
	s, _ := spec.New(10, spec.ECMedium)
	sym, err := qr.Build([]byte("hello"), s)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	n := s.Version.Size()

	// Assert: at every finder origin, check the 4 distinctive bits
	// (NW corner = 1, NW inner adjacent = 0, centre = 1).
	for _, o := range s.Version.FinderOrigins() {
		r, c := o[0], o[1]
		if sym.Grid[r][c] != 1 {
			t.Errorf("finder NW corner (%d,%d) = 0, want 1", r, c)
		}
		if sym.Grid[r+1][c+1] != 0 {
			t.Errorf("finder ring (%d,%d) = 1, want 0", r+1, c+1)
		}
		if sym.Grid[r+3][c+3] != 1 {
			t.Errorf("finder centre (%d,%d) = 0, want 1", r+3, c+3)
		}
	}
	_ = n
}

func TestBuild_TimingPatternsAlternate(t *testing.T) {
	// Arrange
	s, _ := spec.New(5, spec.ECMedium)
	sym, err := qr.Build([]byte("hi"), s)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	n := s.Version.Size()

	// Assert: row 6 and col 6 alternate 1,0,1,0,... in the inner range.
	for k := 8; k < n-8; k++ {
		want := byte(1 - (k % 2))
		if sym.Grid[6][k] != want {
			t.Errorf("timing row[6][%d] = %d, want %d", k, sym.Grid[6][k], want)
		}
		if sym.Grid[k][6] != want {
			t.Errorf("timing col[%d][6] = %d, want %d", k, sym.Grid[k][6], want)
		}
	}
}

func TestBuild_DarkModuleAlwaysOne(t *testing.T) {
	for _, v := range []spec.Version{1, 7, 20, 40} {
		s, _ := spec.New(v, spec.ECMedium)
		sym, err := qr.Build([]byte("x"), s)
		if err != nil {
			t.Fatalf("Build(%s): %v", s, err)
		}
		dr, dc := s.Version.DarkModule()
		if sym.Grid[dr][dc] != 1 {
			t.Errorf("%s: dark module (%d,%d) = 0, want 1", s, dr, dc)
		}
	}
}

func TestBuild_MaskInRange(t *testing.T) {
	s, _ := spec.New(10, spec.ECHigh)
	sym, err := qr.Build([]byte("https://lunar.app"), s)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	if sym.Mask < 0 || sym.Mask >= qr.MaskCount {
		t.Errorf("Mask = %d, want in [0, %d)", sym.Mask, qr.MaskCount)
	}
}

func TestBuild_PayloadOverflowReturnsError(t *testing.T) {
	s, _ := spec.New(1, spec.ECHigh) // V1-H = 7 bytes max
	_, err := qr.Build(make([]byte, 100), s)
	if err == nil {
		t.Fatal("Build(oversized): want error, got nil")
	}
}

func TestEncodeBytes_LengthEqualsTotalCodewords(t *testing.T) {
	for _, v := range []spec.Version{1, 10, 20, 40} {
		for _, ec := range []spec.ECLevel{spec.ECLow, spec.ECMedium, spec.ECQuartile, spec.ECHigh} {
			s, _ := spec.New(v, ec)
			payload := make([]byte, s.MaxByteModePayload()/2)
			cw, err := qr.EncodeBytes(payload, s)
			if err != nil {
				t.Fatalf("%s: %v", s, err)
			}
			if got, want := len(cw), s.TotalCodewords(); got != want {
				t.Errorf("%s: len(EncodeBytes) = %d, want %d", s, got, want)
			}
		}
	}
}

func TestBitStream_LengthIncludesRemainderBits(t *testing.T) {
	for _, v := range []spec.Version{1, 5, 14, 21, 28, 40} {
		s, _ := spec.New(v, spec.ECMedium)
		bits, err := qr.BitStream([]byte("x"), s)
		if err != nil {
			t.Fatalf("V%d: %v", v, err)
		}
		want := s.TotalCodewords()*8 + qr.RemainderBits(v)
		if len(bits) != want {
			t.Errorf("V%d: len(BitStream) = %d, want %d", v, len(bits), want)
		}
	}
}
