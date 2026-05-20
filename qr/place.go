package qr

import (
	"fmt"

	"github.com/rumo-lunar/qrlogo/qr/sym"
)

// PlaceCodewords lays out the 404 V11 interleaved codewords (3232
// bits) into the data region of a 61×61 grid following the QR
// zig-zag traversal defined in ISO/IEC 18004 §7.7.3.
//
// Traversal rules:
//
//   - The matrix is swept right-to-left in 2-column-wide strips
//     starting at column 60.
//   - The vertical timing column (column 6) is skipped entirely:
//     once the loop variable would land on column 6 it jumps to
//     column 5.
//   - Within each strip, modules are visited row by row; the right
//     column is visited before the left column at the same row.
//     The vertical sweep direction alternates per strip — first
//     strip (60, 59) goes bottom-to-top, next (58, 57) top-to-
//     bottom, and so on.
//   - Cells whose Kind is not KindData are skipped silently. The
//     codeword bit cursor only advances when a data cell receives a
//     bit.
//
// Codeword bits are emitted MSB-first within each byte, so
// codeword[k] bit 0 lands at the first data cell visited after
// k·8 prior bits have already been placed.
//
// The returned grid has dimensions m.Size × m.Size. Non-data cells
// hold the zero Bit (no variables, Const = 0); callers must consult
// m to know which cells are which.
//
// Panics if the codeword count is wrong or the placement does not
// consume exactly Size² − (function-pattern modules) bits — both of
// which would indicate a bug in the placement loop or in the map.
func PlaceCodewords(d *sym.Domain, m *Map, codewords []sym.Byte) [][]sym.Bit {
	want := DataCodewordsV11M + ECCodewordsV11M
	if len(codewords) != want {
		panic(fmt.Sprintf("qr.PlaceCodewords: got %d codewords, want %d",
			len(codewords), want))
	}

	n := m.Size
	grid := make([][]sym.Bit, n)
	zero := d.ConstBit(0)
	for r := range grid {
		grid[r] = make([]sym.Bit, n)
		for c := range grid[r] {
			grid[r][c] = zero
		}
	}

	bitIdx := 0
	upward := true
	for col := n - 1; col > 0; col -= 2 {
		if col == 6 {
			// Skip the vertical timing column entirely.
			col = 5
		}
		if upward {
			for r := n - 1; r >= 0; r-- {
				placeOne(m, grid, r, col, codewords, &bitIdx)
				placeOne(m, grid, r, col-1, codewords, &bitIdx)
			}
		} else {
			for r := 0; r < n; r++ {
				placeOne(m, grid, r, col, codewords, &bitIdx)
				placeOne(m, grid, r, col-1, codewords, &bitIdx)
			}
		}
		upward = !upward
	}

	if expected := want * 8; bitIdx != expected {
		panic(fmt.Sprintf("qr.PlaceCodewords: placed %d bits, want %d",
			bitIdx, expected))
	}
	return grid
}

// placeOne deposits one bit at (r, c) iff the cell is a data cell.
func placeOne(m *Map, grid [][]sym.Bit, r, c int, cw []sym.Byte, bitIdx *int) {
	if m.KindAt(r, c) != KindData {
		return
	}
	grid[r][c] = cw[*bitIdx/8][*bitIdx%8]
	*bitIdx++
}
