package qr

import (
	"github.com/rumo-lunar/qrlogo/qr/sym"
)

// ApplyMask2 returns a new ghost grid with QR data-mask pattern 2
// applied to a V40 symbol. The mask flips a data-region bit whenever
// its column index is divisible by 3:
//
//	mask₂(row, col) = 1 ⟺ col mod 3 == 0
//
// Per ISO/IEC 18004 §7.8.2, the mask is XORed only with data-region
// modules (KindData cells); function-pattern cells are untouched.
//
// Because the mask is a constant boolean indexed by position, "flip"
// becomes a single XOR into the Const term of each affected Bit;
// variable dependencies are preserved unchanged. Masking is an
// involution: ApplyMask2(ApplyMask2(g, m), m) == g cell-by-cell.
//
// The returned grid is a fresh allocation: the input grid is not
// mutated.
func ApplyMask2(d *sym.Domain, m *Map, grid [][]sym.Bit) [][]sym.Bit {
	out := make([][]sym.Bit, m.Size)
	for r := range out {
		out[r] = make([]sym.Bit, m.Size)
		copy(out[r], grid[r])
	}
	one := d.ConstBit(1)
	for r := 0; r < m.Size; r++ {
		for c := 0; c < m.Size; c++ {
			if m.KindAt(r, c) != KindData {
				continue
			}
			if c%3 != 0 {
				continue
			}
			out[r][c] = d.XorBit(out[r][c], one)
		}
	}
	return out
}
