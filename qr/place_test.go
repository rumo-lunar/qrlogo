package qr_test

import (
	"math/rand"
	"strings"
	"testing"

	"github.com/rumo-lunar/qrlogo/qr"
	"github.com/rumo-lunar/qrlogo/qr/sym"
)

// referencePlace is the concrete-byte mirror of PlaceCodewords. It
// runs the identical zig-zag traversal but writes raw bits from a
// concrete []byte codeword stream.
func referencePlace(m *qr.Map, codewords []byte) [][]byte {
	n := m.Size
	grid := make([][]byte, n)
	for r := range grid {
		grid[r] = make([]byte, n)
	}
	bitIdx := 0
	upward := true
	for col := n - 1; col > 0; col -= 2 {
		if col == 6 {
			col = 5
		}
		if upward {
			for r := n - 1; r >= 0; r-- {
				placeOneConcrete(m, grid, r, col, codewords, &bitIdx)
				placeOneConcrete(m, grid, r, col-1, codewords, &bitIdx)
			}
		} else {
			for r := 0; r < n; r++ {
				placeOneConcrete(m, grid, r, col, codewords, &bitIdx)
				placeOneConcrete(m, grid, r, col-1, codewords, &bitIdx)
			}
		}
		upward = !upward
	}
	return grid
}

func placeOneConcrete(m *qr.Map, grid [][]byte, r, c int, cw []byte, bitIdx *int) {
	if m.KindAt(r, c) != qr.KindData {
		return
	}
	grid[r][c] = (cw[*bitIdx/8] >> uint(7-*bitIdx%8)) & 1
	*bitIdx++
}

func TestPlaceCodewords_GridShape(t *testing.T) {
	// Arrange
	d := sym.NewDomain(0)
	cw := make([]sym.Byte, qr.DataCodewordsV11M+qr.ECCodewordsV11M)
	zero := d.ConstByte(0)
	for i := range cw {
		cw[i] = zero
	}
	m := qr.NewV11Map()

	// Act
	grid := qr.PlaceCodewords(d, m, cw)

	// Assert
	if len(grid) != 61 {
		t.Fatalf("rows = %d, want 61", len(grid))
	}
	for r, row := range grid {
		if len(row) != 61 {
			t.Errorf("row %d width = %d, want 61", r, len(row))
		}
	}
}

func TestPlaceCodewords_FirstCodewordLandsAtBottomRight(t *testing.T) {
	// Arrange: codeword 0 carries variables 0..7 (MSB-first), every
	// other codeword is constant 0. So a solution that flips x_i
	// should flip exactly one cell — the placement target of bit i
	// of codeword 0.
	const numVars = 8
	d := sym.NewDomain(numVars)
	cw := make([]sym.Byte, qr.DataCodewordsV11M+qr.ECCodewordsV11M)
	zero := d.ConstByte(0)
	for i := range cw {
		cw[i] = zero
	}
	var first sym.Byte
	for j := 0; j < 8; j++ {
		first[j] = d.Variable(j)
	}
	cw[0] = first
	m := qr.NewV11Map()

	// Act
	grid := qr.PlaceCodewords(d, m, cw)

	// Assert: the zig-zag starts at (60, 60) going upward. (60, 60)
	// and (60, 59) are both free data cells (no function pattern
	// covers them) and receive bits 0 and 1 of codeword 0.
	//
	// Continuing upward: (59, 60), (59, 59), (58, 60), (58, 59),
	// (57, 60), (57, 59) all KindData → bits 2..7 of codeword 0.
	wantPositions := []struct {
		row, col int
		varIdx   int
	}{
		{60, 60, 0}, // MSB of codeword 0
		{60, 59, 1},
		{59, 60, 2},
		{59, 59, 3},
		{58, 60, 4},
		{58, 59, 5},
		{57, 60, 6},
		{57, 59, 7}, // LSB of codeword 0
	}
	for _, wp := range wantPositions {
		// Solution flipping only x_{wp.varIdx}.
		sol := []byte{1 << uint(7-wp.varIdx)}
		if got := d.ResolveBit(grid[wp.row][wp.col], sol); got != 1 {
			t.Errorf("flipping x_%d: grid[%d][%d] = %d, want 1",
				wp.varIdx, wp.row, wp.col, got)
		}
	}
}

func TestPlaceCodewords_NonDataCellsAreZero(t *testing.T) {
	// Arrange: every codeword bit is variable 0. Then any cell that
	// got a placed bit will depend on x_0; any cell that didn't get
	// one stays at constant 0 and is independent of x_0.
	d := sym.NewDomain(1)
	cw := make([]sym.Byte, qr.DataCodewordsV11M+qr.ECCodewordsV11M)
	var all sym.Byte
	for j := 0; j < 8; j++ {
		all[j] = d.Variable(0)
	}
	for i := range cw {
		cw[i] = all
	}
	m := qr.NewV11Map()

	// Act
	grid := qr.PlaceCodewords(d, m, cw)

	// Assert: with x_0 = 1, every KindData cell resolves to 1; every
	// other Kind resolves to 0.
	sol := []byte{0x80} // x_0 = 1
	for r := 0; r < m.Size; r++ {
		for c := 0; c < m.Size; c++ {
			got := d.ResolveBit(grid[r][c], sol)
			if m.KindAt(r, c) == qr.KindData {
				if got != 1 {
					t.Errorf("data cell (%d,%d) = %d, want 1", r, c, got)
				}
			} else {
				if got != 0 {
					t.Errorf("non-data cell (%d,%d) Kind=%v = %d, want 0",
						r, c, m.KindAt(r, c), got)
				}
			}
		}
	}
}

func TestPlaceCodewords_FullPipelineLinearity(t *testing.T) {
	// Arrange: real URL through EncodeData → InterleaveV11M →
	// PlaceCodewords. Resolve every grid cell against random
	// solutions and compare against the concrete reference placement.
	url := strings.Repeat("p", 80)
	data, d := qr.EncodeData(url)
	cw := qr.InterleaveV11M(d, data)
	m := qr.NewV11Map()
	symGrid := qr.PlaceCodewords(d, m, cw)

	rng := rand.New(rand.NewSource(20260520))
	for trial := 0; trial < 3; trial++ {
		sol := make([]byte, (d.NumVars+7)/8)
		for i := range sol {
			sol[i] = byte(rng.Intn(256))
		}

		// Resolve the codeword stream to concrete bytes.
		concrete := make([]byte, len(cw))
		for i := range cw {
			concrete[i] = d.ResolveByte(cw[i], sol)
		}
		refGrid := referencePlace(m, concrete)

		// Compare cell-by-cell.
		for r := 0; r < m.Size; r++ {
			for c := 0; c < m.Size; c++ {
				got := d.ResolveBit(symGrid[r][c], sol)
				want := refGrid[r][c]
				if got != want {
					t.Fatalf("trial %d (%d,%d) Kind=%v: got %d, want %d",
						trial, r, c, m.KindAt(r, c), got, want)
				}
			}
		}
	}
}

func TestPlaceCodewords_PanicsOnWrongCount(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("expected panic on wrong codeword count")
		}
	}()
	d := sym.NewDomain(0)
	m := qr.NewV11Map()
	qr.PlaceCodewords(d, m, make([]sym.Byte, 100))
}
