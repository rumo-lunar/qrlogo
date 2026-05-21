package qr_test

import (
	"testing"

	"github.com/rumo-lunar/qrlogo/qr"
)

func TestFunctionBits_Shape(t *testing.T) {
	// Arrange + Act
	g := qr.FunctionBits()

	// Assert
	if len(g) != 177 {
		t.Fatalf("rows = %d, want 177", len(g))
	}
	for r, row := range g {
		if len(row) != 177 {
			t.Errorf("row %d width = %d, want 177", r, len(row))
		}
	}
}

func TestFunctionBits_DarkModule(t *testing.T) {
	// Arrange + Act
	g := qr.FunctionBits()

	// Assert: dark module at (4*40+9, 8) = (169, 8) must be 1.
	if got := g[169][8]; got != 1 {
		t.Errorf("dark module (169,8) = %d, want 1", got)
	}
}

func TestFunctionBits_FinderPattern(t *testing.T) {
	// Arrange + Act
	g := qr.FunctionBits()
	n := 177

	// Assert
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
		// Top-right finder (cols n-7..n-1 = 170..176)
		{0, n - 7, 1, "TR outer left"},
		{0, n - 1, 1, "TR outer right"},
		{3, n - 4, 1, "TR centre"},
		// Bottom-left finder (rows n-7..n-1 = 170..176)
		{n - 7, 0, 1, "BL outer top"},
		{n - 1, 0, 1, "BL outer bottom"},
		{n - 4, 3, 1, "BL centre"},
	}
	for _, c := range cases {
		if got := g[c.row][c.col]; got != c.want {
			t.Errorf("(%d,%d) [%s] = %d, want %d",
				c.row, c.col, c.note, got, c.want)
		}
	}
}

func TestFunctionBits_SeparatorsAreLight(t *testing.T) {
	// Arrange + Act
	g := qr.FunctionBits()
	n := 177

	// Assert: TL separator row and col
	cases := [][2]int{
		{7, 0}, {7, 7}, // TL separator row
		{0, 7}, {6, 7}, // TL separator col
		// TR separator
		{7, n - 8}, {7, n - 1}, // TR separator row
		{0, n - 8}, {6, n - 8}, // TR separator col
		// BL separator
		{n - 8, 0}, {n - 8, 7}, // BL separator row
		{n - 7, 7}, {n - 1, 7}, // BL separator col
	}
	for _, c := range cases {
		if got := g[c[0]][c[1]]; got != 0 {
			t.Errorf("separator (%d,%d) = %d, want 0", c[0], c[1], got)
		}
	}
}

func TestFunctionBits_TimingPattern(t *testing.T) {
	// Arrange + Act
	g := qr.FunctionBits()

	// Assert: timing on row 6 between finders. Module col 8 is dark (even).
	cases := []struct {
		row, col int
		want     byte
		note     string
	}{
		{6, 8, 1, "h-timing col=8 (dark)"},
		{6, 9, 0, "h-timing col=9 (light)"},
		{6, 10, 1, "h-timing col=10 (dark)"},
		{8, 6, 1, "v-timing row=8 (dark)"},
		{9, 6, 0, "v-timing row=9 (light)"},
	}
	for _, c := range cases {
		if got := g[c.row][c.col]; got != c.want {
			t.Errorf("(%d,%d) [%s] = %d, want %d",
				c.row, c.col, c.note, got, c.want)
		}
	}
}

func TestFunctionBits_AlignmentPattern(t *testing.T) {
	// Arrange + Act
	g := qr.FunctionBits()

	// Assert: centre alignment at (86,86) should have a dark centre.
	cases := []struct {
		row, col int
		want     byte
		note     string
	}{
		{86, 86, 1, "centre (86,86) alignment centre dark"},
		{85, 86, 0, "alignment white ring"},
		{84, 84, 1, "alignment outer corner"},
		{88, 88, 1, "alignment outer corner"},
		// An alignment at last valid: (142,170)
		{142, 170, 1, "alignment (142,170) centre"},
		{140, 168, 1, "alignment (142,170) outer corner"},
	}
	for _, c := range cases {
		if got := g[c.row][c.col]; got != c.want {
			t.Errorf("(%d,%d) [%s] = %d, want %d",
				c.row, c.col, c.note, got, c.want)
		}
	}
}

func TestFunctionBits_DataCellsAreZero(t *testing.T) {
	// Arrange
	g := qr.FunctionBits()
	m := qr.NewMap()

	// Assert: all KindData cells in function grid must be zero-filled placeholders.
	for r := 0; r < m.Size; r++ {
		for c := 0; c < m.Size; c++ {
			if m.KindAt(r, c) != qr.KindData {
				continue
			}
			if g[r][c] != 0 {
				t.Errorf("data cell (%d,%d) = %d, want 0 (placeholder)", r, c, g[r][c])
			}
		}
	}
}

func TestNewMap_DataModuleCount(t *testing.T) {
	// Arrange + Act
	m := qr.NewMap()

	// Assert: data module count must be exactly 3706*8 = 29648.
	dataCount := 0
	for r := 0; r < m.Size; r++ {
		for c := 0; c < m.Size; c++ {
			if m.KindAt(r, c) == qr.KindData {
				dataCount++
			}
		}
	}
	want := (qr.DataCodewords + qr.ECCodewords) * 8
	if dataCount != want {
		t.Errorf("data modules = %d, want %d", dataCount, want)
	}
}

func TestFunctionBits_FormatConstant(t *testing.T) {
	// Assert: FormatMMask2 has the expected literal bit pattern.
	if qr.FormatMMask2 != 0b101111001111100 {
		t.Errorf("FormatMMask2 = 0x%04x, want 0x%04x",
			qr.FormatMMask2, 0b101111001111100)
	}
}

func TestFunctionBits_VersionConstant(t *testing.T) {
	if qr.VersionInfo != 0b101000110001101001 {
		t.Errorf("VersionInfo = 0x%06x, want 0x%06x",
			qr.VersionInfo, 0b101000110001101001)
	}
}
