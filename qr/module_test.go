package qr_test

import (
	"testing"

	"github.com/rumo-lunar/qrlogo/qr"
)

func TestNewV11Map_Size(t *testing.T) {
	// Arrange + Act
	sut := qr.NewV11Map()

	// Assert
	if sut.Size != 61 {
		t.Fatalf("expected Size=61, got %d", sut.Size)
	}
}

func TestNewV11Map_FreeDataCount(t *testing.T) {
	// Arrange + Act
	sut := qr.NewV11Map()

	// Assert: V11 has 404 codewords × 8 = 3232 data bits, 0 remainder.
	free := 0
	for r := 0; r < sut.Size; r++ {
		for c := 0; c < sut.Size; c++ {
			if sut.KindAt(r, c) == qr.KindData {
				free++
			}
		}
	}
	if free != 3232 {
		t.Fatalf("expected 3232 free data modules, got %d", free)
	}
}

func TestNewV11Map_FunctionPatternCounts(t *testing.T) {
	// Arrange + Act
	sut := qr.NewV11Map()

	// Assert per-kind counts.
	counts := map[qr.Kind]int{}
	for r := 0; r < sut.Size; r++ {
		for c := 0; c < sut.Size; c++ {
			counts[sut.KindAt(r, c)]++
		}
	}

	// Expected values derived from the V11 spec:
	//   Finder:    3 × 7×7 = 147
	//   Separator: 3 × 15   = 45     (row + col, sharing the corner cell)
	//   Alignment: 6 × 5×5  = 150
	//   Timing:    2 × 20   = 40  ⇒ 80 total (row 6 and col 6 combined)
	//   Version:   2 × 18   = 36
	//   Format:    15 + 15  = 30
	//   Dark:      1
	//   ───────────────────────
	//                       = 489 function modules
	//   Data = 61² − 489    = 3232
	want := map[qr.Kind]int{
		qr.KindFinder:    147,
		qr.KindSeparator: 45,
		qr.KindAlignment: 150,
		qr.KindTiming:    80,
		qr.KindVersion:   36,
		qr.KindFormat:    30,
		qr.KindDark:      1,
		qr.KindData:      3232,
	}
	for k, w := range want {
		if got := counts[k]; got != w {
			t.Errorf("%v: got %d, want %d", k, got, w)
		}
	}

	// Sanity: counts must sum to 61² = 3721.
	total := 0
	for _, n := range counts {
		total += n
	}
	if total != 3721 {
		t.Errorf("total modules = %d, want 3721", total)
	}
}

func TestNewV11Map_SpotPositions(t *testing.T) {
	// Arrange + Act
	sut := qr.NewV11Map()

	// Assert: known-position spot checks across every Kind.
	cases := []struct {
		row, col int
		want     qr.Kind
		note     string
	}{
		{0, 0, qr.KindFinder, "top-left finder outer corner"},
		{3, 3, qr.KindFinder, "top-left finder centre"},
		{0, 60, qr.KindFinder, "top-right finder outer corner"},
		{60, 0, qr.KindFinder, "bottom-left finder outer corner"},

		{7, 0, qr.KindSeparator, "top-left separator (row 7, data side)"},
		{0, 7, qr.KindSeparator, "top-left separator (col 7, data side)"},
		{7, 7, qr.KindSeparator, "top-left separator (corner of L)"},
		{7, 53, qr.KindSeparator, "top-right separator (col 53)"},
		{53, 7, qr.KindSeparator, "bottom-left separator (row 53)"},

		{6, 10, qr.KindTiming, "row-6 timing strip"},
		{10, 6, qr.KindTiming, "col-6 timing strip"},

		{30, 30, qr.KindAlignment, "centre alignment pattern centre"},
		{28, 28, qr.KindAlignment, "centre alignment outer ring"},
		{56, 56, qr.KindAlignment, "bottom-right alignment centre"},

		{53, 8, qr.KindDark, "dark module at (4·11+9, 8)"},

		{8, 0, qr.KindFormat, "format strip A (row 8 lower segment)"},
		{0, 8, qr.KindFormat, "format strip A (col 8 upper segment)"},
		{8, 60, qr.KindFormat, "format strip B (row 8 right segment)"},
		{60, 8, qr.KindFormat, "format strip B (col 8 bottom segment)"},

		{0, 50, qr.KindVersion, "version block A (above bottom-left finder, top row)"},
		{5, 52, qr.KindVersion, "version block A (above bottom-left finder, bottom-right cell)"},
		{50, 0, qr.KindVersion, "version block B (left of top-right finder, top-left cell)"},
		{52, 5, qr.KindVersion, "version block B (left of top-right finder, bottom-right cell)"},

		{60, 60, qr.KindData, "bottom-right corner is free data (no finder there)"},
		{9, 9, qr.KindData, "interior data cell"},
	}
	for _, c := range cases {
		if got := sut.KindAt(c.row, c.col); got != c.want {
			t.Errorf("KindAt(%d,%d) = %v, want %v (%s)",
				c.row, c.col, got, c.want, c.note)
		}
	}
}
