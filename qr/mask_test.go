package qr_test

import (
	"math/rand"
	"strings"
	"testing"

	"github.com/rumo-lunar/qrlogo/qr"
	"github.com/rumo-lunar/qrlogo/qr/sym"
)

// zeroGrid returns a 61×61 ghost grid whose cells are all the
// constant 0 bit. Mask 2 applied to this grid produces a grid that
// resolves to mask₂(r, c) on KindData cells and 0 elsewhere.
func zeroGrid(d *sym.Domain, size int) [][]sym.Bit {
	zero := d.ConstBit(0)
	g := make([][]sym.Bit, size)
	for r := range g {
		g[r] = make([]sym.Bit, size)
		for c := range g[r] {
			g[r][c] = zero
		}
	}
	return g
}

func TestApplyMask2_GridShapePreserved(t *testing.T) {
	// Arrange
	d := sym.NewDomain(0)
	m := qr.NewV11Map()
	in := zeroGrid(d, m.Size)

	// Act
	out := qr.ApplyMask2(d, m, in)

	// Assert
	if len(out) != m.Size {
		t.Fatalf("rows = %d, want %d", len(out), m.Size)
	}
	for r, row := range out {
		if len(row) != m.Size {
			t.Errorf("row %d width = %d, want %d", r, len(row), m.Size)
		}
	}
}

func TestApplyMask2_FlipsExactlyDataColumnsDivisibleBy3(t *testing.T) {
	// Arrange: starting from an all-zero grid, after mask 2 each
	// KindData cell at col%3 == 0 should resolve to 1; all other
	// cells (other KindData and non-data) should resolve to 0.
	d := sym.NewDomain(0)
	m := qr.NewV11Map()
	in := zeroGrid(d, m.Size)

	// Act
	out := qr.ApplyMask2(d, m, in)

	// Assert
	sol := []byte{}
	for r := 0; r < m.Size; r++ {
		for c := 0; c < m.Size; c++ {
			got := d.ResolveBit(out[r][c], sol)
			isDataMaskedCell := m.KindAt(r, c) == qr.KindData && c%3 == 0
			var want byte
			if isDataMaskedCell {
				want = 1
			}
			if got != want {
				t.Errorf("(%d,%d) Kind=%v: got %d, want %d",
					r, c, m.KindAt(r, c), got, want)
			}
		}
	}
}

func TestApplyMask2_NonDataCellsUntouched(t *testing.T) {
	// Arrange: a grid where every data cell is variable 0, every
	// other cell is the zero bit. After masking, non-data cells
	// must still resolve to 0 for any value of x_0.
	d := sym.NewDomain(1)
	m := qr.NewV11Map()
	one := d.Variable(0)
	zero := d.ConstBit(0)
	g := make([][]sym.Bit, m.Size)
	for r := range g {
		g[r] = make([]sym.Bit, m.Size)
		for c := 0; c < m.Size; c++ {
			if m.KindAt(r, c) == qr.KindData {
				g[r][c] = one
			} else {
				g[r][c] = zero
			}
		}
	}

	// Act
	out := qr.ApplyMask2(d, m, g)

	// Assert: non-data cells resolve to 0 regardless of x_0.
	for _, solByte := range []byte{0x00, 0x80} {
		sol := []byte{solByte}
		for r := 0; r < m.Size; r++ {
			for c := 0; c < m.Size; c++ {
				if m.KindAt(r, c) == qr.KindData {
					continue
				}
				if got := d.ResolveBit(out[r][c], sol); got != 0 {
					t.Errorf("non-data (%d,%d) Kind=%v: got %d, want 0",
						r, c, m.KindAt(r, c), got)
				}
			}
		}
	}
}

func TestApplyMask2_IsInvolution(t *testing.T) {
	// Arrange: run the full symbolic pipeline up to PlaceCodewords
	// (so we have a realistically populated grid with constants and
	// linear forms), then apply mask twice and compare.
	url := strings.Repeat("k", 60)
	data, d := qr.EncodeData(url)
	cw := qr.InterleaveV11M(d, data)
	m := qr.NewV11Map()
	g0 := qr.PlaceCodewords(d, m, cw)

	// Act
	g1 := qr.ApplyMask2(d, m, g0)
	g2 := qr.ApplyMask2(d, m, g1)

	// Assert: resolved values of g2 must match g0 on every cell for
	// every solution we try.
	rng := rand.New(rand.NewSource(7))
	for trial := 0; trial < 3; trial++ {
		sol := make([]byte, (d.NumVars+7)/8)
		for i := range sol {
			sol[i] = byte(rng.Intn(256))
		}
		for r := 0; r < m.Size; r++ {
			for c := 0; c < m.Size; c++ {
				if d.ResolveBit(g0[r][c], sol) != d.ResolveBit(g2[r][c], sol) {
					t.Fatalf("trial %d: mask is not involution at (%d,%d)",
						trial, r, c)
				}
			}
		}
	}
}

func TestApplyMask2_PreservesVariableDependencies(t *testing.T) {
	// Arrange: the mask only flips the Const term; the variable set
	// of any affected Bit must be unchanged. Build a grid where every
	// data cell at col 0 (which is divisible by 3) is a known
	// variable form, apply mask, and check the resolution under two
	// solutions that differ only in that variable still differ by 1.
	const numVars = 1
	d := sym.NewDomain(numVars)
	m := qr.NewV11Map()
	v := d.Variable(0)
	zero := d.ConstBit(0)
	g := make([][]sym.Bit, m.Size)
	for r := range g {
		g[r] = make([]sym.Bit, m.Size)
		for c := 0; c < m.Size; c++ {
			if m.KindAt(r, c) == qr.KindData && c == 0 {
				g[r][c] = v
			} else {
				g[r][c] = zero
			}
		}
	}

	// Act
	out := qr.ApplyMask2(d, m, g)

	// Assert: pick any data cell at col 0 — say (9, 0) which is
	// KindData (it's not in any function pattern). With x_0=0 it
	// resolves to 1 (the mask flip on top of v=0). With x_0=1 it
	// resolves to 0 (the mask flip on top of v=1). So the resolutions
	// must differ between the two solutions.
	if m.KindAt(9, 0) != qr.KindData {
		t.Fatalf("test assumption broken: (9,0) is %v", m.KindAt(9, 0))
	}
	got0 := d.ResolveBit(out[9][0], []byte{0x00})
	got1 := d.ResolveBit(out[9][0], []byte{0x80})
	if got0 != 1 {
		t.Errorf("(9,0) with x_0=0: got %d, want 1 (= 0 ⊕ 1)", got0)
	}
	if got1 != 0 {
		t.Errorf("(9,0) with x_0=1: got %d, want 0 (= 1 ⊕ 1)", got1)
	}
}
