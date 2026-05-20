package qr_test

import (
	"testing"

	"github.com/rumo-lunar/qrlogo/qr"
)

func TestFunctionBitsV11M_Shape(t *testing.T) {
	g := qr.FunctionBitsV11M()
	if len(g) != 61 {
		t.Fatalf("rows = %d, want 61", len(g))
	}
	for r, row := range g {
		if len(row) != 61 {
			t.Errorf("row %d width = %d, want 61", r, len(row))
		}
	}
}

func TestFunctionBitsV11M_FinderPattern(t *testing.T) {
	g := qr.FunctionBitsV11M()

	// Standard finder pattern reused for all three corners.
	cases := []struct {
		row, col int
		want     byte
		note     string
	}{
		// Top-left finder
		{0, 0, 1, "TL outer corner"},
		{6, 6, 1, "TL inner corner"},
		{1, 1, 0, "TL white ring"},
		{3, 3, 1, "TL centre"},
		{2, 4, 1, "TL centre 3x3 right edge"},
		// Top-right finder (cols 54..60)
		{0, 54, 1, "TR outer left"},
		{0, 60, 1, "TR outer right"},
		{1, 55, 0, "TR white ring"},
		{3, 57, 1, "TR centre"},
		// Bottom-left finder (rows 54..60)
		{54, 0, 1, "BL outer top"},
		{60, 0, 1, "BL outer bottom"},
		{55, 1, 0, "BL white ring"},
		{57, 3, 1, "BL centre"},
	}
	for _, c := range cases {
		if got := g[c.row][c.col]; got != c.want {
			t.Errorf("(%d,%d) [%s] = %d, want %d",
				c.row, c.col, c.note, got, c.want)
		}
	}
}

func TestFunctionBitsV11M_SeparatorsAreLight(t *testing.T) {
	g := qr.FunctionBitsV11M()
	cases := [][2]int{
		{7, 0}, {7, 5}, {7, 7},  // TL separator
		{0, 7}, {6, 7},          // TL separator col
		{7, 53}, {7, 60},        // TR separator row
		{0, 53}, {6, 53},        // TR separator col
		{53, 0}, {53, 7},        // BL separator row
		{54, 7}, {60, 7},        // BL separator col
	}
	for _, c := range cases {
		if got := g[c[0]][c[1]]; got != 0 {
			t.Errorf("separator (%d,%d) = %d, want 0", c[0], c[1], got)
		}
	}
}

func TestFunctionBitsV11M_TimingPattern(t *testing.T) {
	g := qr.FunctionBitsV11M()

	// Row 6 horizontal timing spans cols 8..52 (skipping alignment 28..32).
	// Dark at even col, light at odd col.
	cases := []struct {
		row, col int
		want     byte
		note     string
	}{
		{6, 8, 1, "h-timing start (dark)"},
		{6, 9, 0, "h-timing next (light)"},
		{6, 10, 1, "h-timing dark"},
		{6, 27, 0, "h-timing just before alignment"},
		{6, 33, 0, "h-timing just after alignment"},
		{6, 52, 1, "h-timing end (dark)"},
		{8, 6, 1, "v-timing start (dark)"},
		{9, 6, 0, "v-timing next (light)"},
		{52, 6, 1, "v-timing end (dark)"},
		// Alignment-overlap cells must NOT be overwritten by timing
		// (they belong to alignment patterns).
		{6, 30, 1, "alignment centre on row 6 (col 30 alignment, value=1)"},
	}
	for _, c := range cases {
		if got := g[c.row][c.col]; got != c.want {
			t.Errorf("(%d,%d) [%s] = %d, want %d",
				c.row, c.col, c.note, got, c.want)
		}
	}
}

func TestFunctionBitsV11M_AlignmentPattern(t *testing.T) {
	g := qr.FunctionBitsV11M()

	// Spot checks on the centre alignment (30, 30) and the
	// bottom-right alignment (56, 56).
	cases := []struct {
		row, col int
		want     byte
		note     string
	}{
		{30, 30, 1, "centre alignment centre dark"},
		{29, 30, 0, "centre alignment white ring"},
		{28, 28, 1, "centre alignment outer corner"},
		{32, 32, 1, "centre alignment outer corner"},
		{56, 56, 1, "BR alignment centre"},
		{54, 54, 1, "BR alignment outer corner"},
	}
	for _, c := range cases {
		if got := g[c.row][c.col]; got != c.want {
			t.Errorf("(%d,%d) [%s] = %d, want %d",
				c.row, c.col, c.note, got, c.want)
		}
	}
}

func TestFunctionBitsV11M_DarkModule(t *testing.T) {
	g := qr.FunctionBitsV11M()
	if got := g[53][8]; got != 1 {
		t.Errorf("dark module (53,8) = %d, want 1", got)
	}
}

func TestFunctionBitsV11M_FormatBits(t *testing.T) {
	g := qr.FunctionBitsV11M()
	// FormatV11MMask2 = 101111001111100 (bits 14..0)
	//   bit 14=1, 13=0, 12=1, 11=1, 10=1, 9=1, 8=0, 7=0,
	//   bit 6=1,  5=1,  4=1,  3=1,  2=1,  1=0, 0=0.
	cases := []struct {
		row, col int
		want     byte
		bit      int
	}{
		// Location 1 (around top-left finder)
		{8, 0, 1, 14}, {8, 1, 0, 13}, {8, 2, 1, 12}, {8, 3, 1, 11},
		{8, 4, 1, 10}, {8, 5, 1, 9}, {8, 7, 0, 8}, {8, 8, 0, 7},
		{7, 8, 1, 6}, {5, 8, 1, 5}, {4, 8, 1, 4}, {3, 8, 1, 3},
		{2, 8, 1, 2}, {1, 8, 0, 1}, {0, 8, 0, 0},
		// Location 2 (bottom-left + top-right)
		{60, 8, 1, 14}, {59, 8, 0, 13}, {58, 8, 1, 12}, {57, 8, 1, 11},
		{56, 8, 1, 10}, {55, 8, 1, 9}, {54, 8, 0, 8},
		{8, 53, 0, 7}, {8, 54, 1, 6}, {8, 55, 1, 5}, {8, 56, 1, 4},
		{8, 57, 1, 3}, {8, 58, 1, 2}, {8, 59, 0, 1}, {8, 60, 0, 0},
	}
	for _, c := range cases {
		if got := g[c.row][c.col]; got != c.want {
			t.Errorf("format bit %d at (%d,%d) = %d, want %d",
				c.bit, c.row, c.col, got, c.want)
		}
	}
}

func TestFunctionBitsV11M_VersionBits(t *testing.T) {
	g := qr.FunctionBitsV11M()
	// VersionV11 = 001011101111110110 (bits 17..0)
	//   bit 17=0, 16=0, 15=1, 14=0, 13=1, 12=1, 11=1, 10=0, 9=1,
	//   bit 8=1,  7=1,  6=1,  5=1,  4=1,  3=0,  2=1, 1=1, 0=0.

	// Block A (top-right area): cell (r, 50+j) holds bit r*3+j.
	caseSet := func(r, c, bitIdx int, want byte) string {
		return ""
	}
	_ = caseSet
	type bitCase struct {
		row, col int
		bitIdx   int
		want     byte
	}
	cases := []bitCase{
		// Block A spot-checks
		{0, 50, 0, 0},
		{0, 51, 1, 1},
		{0, 52, 2, 1},
		{1, 50, 3, 0},
		{1, 52, 5, 1},
		{2, 50, 6, 1},
		{2, 51, 7, 1},
		{2, 52, 8, 1},
		{3, 51, 10, 0},
		{5, 50, 15, 1},
		{5, 51, 16, 0},
		{5, 52, 17, 0},
		// Block B spot-checks
		{50, 0, 0, 0},
		{51, 0, 1, 1},
		{52, 0, 2, 1},
		{50, 1, 3, 0},
		{51, 1, 4, 1},
		{52, 1, 5, 1},
		{50, 5, 15, 1},
		{51, 5, 16, 0},
		{52, 5, 17, 0},
	}
	for _, c := range cases {
		if got := g[c.row][c.col]; got != c.want {
			t.Errorf("version bit %d at (%d,%d) = %d, want %d",
				c.bitIdx, c.row, c.col, got, c.want)
		}
	}
}

func TestFunctionBitsV11M_DataCellsAreZero(t *testing.T) {
	g := qr.FunctionBitsV11M()
	m := qr.NewV11Map()
	for r := 0; r < m.Size; r++ {
		for c := 0; c < m.Size; c++ {
			if m.KindAt(r, c) != qr.KindData {
				continue
			}
			if g[r][c] != 0 {
				t.Errorf("data cell (%d,%d) = %d, want 0 (placeholder)",
					r, c, g[r][c])
			}
		}
	}
}

func TestFunctionBitsV11M_FormatConstant(t *testing.T) {
	// Sanity check: the literal FormatV11MMask2 value matches its
	// documented bit pattern. Catches typos in the constant.
	if qr.FormatV11MMask2 != 0b101111001111100 {
		t.Errorf("FormatV11MMask2 = 0x%04x, want 0x%04x",
			qr.FormatV11MMask2, 0b101111001111100)
	}
}

func TestFunctionBitsV11M_VersionConstant(t *testing.T) {
	if qr.VersionV11 != 0b001011101111110110 {
		t.Errorf("VersionV11 = 0x%05x, want 0x%05x",
			qr.VersionV11, 0b001011101111110110)
	}
}
