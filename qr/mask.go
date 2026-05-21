package qr

import "github.com/rumo-lunar/qrlogo/qr/spec"

// MaskCount is the number of data-mask patterns defined by
// ISO/IEC 18004 §7.8.2.
const MaskCount = 8

// maskBit returns 1 iff data cell (r, c) should be flipped under
// mask m, per ISO/IEC 18004 §7.8.2. m ∈ [0, 7].
func maskBit(m, r, c int) byte {
	switch m {
	case 0:
		return boolBit((r+c)%2 == 0)
	case 1:
		return boolBit(r%2 == 0)
	case 2:
		return boolBit(c%3 == 0)
	case 3:
		return boolBit((r+c)%3 == 0)
	case 4:
		return boolBit((r/2+c/3)%2 == 0)
	case 5:
		return boolBit((r*c)%2+(r*c)%3 == 0)
	case 6:
		return boolBit(((r*c)%2+(r*c)%3)%2 == 0)
	case 7:
		return boolBit(((r+c)%2+(r*c)%3)%2 == 0)
	}
	panic("qr.maskBit: invalid mask")
}

func boolBit(b bool) byte {
	if b {
		return 1
	}
	return 0
}

// ApplyMask XORs the mask pattern m onto every data module of grid
// in place. Function-pattern cells (per kinds) are left untouched.
func ApplyMask(grid [][]byte, kinds *Map, m int) {
	n := kinds.Size
	for r := 0; r < n; r++ {
		for c := 0; c < n; c++ {
			if kinds.IsData(r, c) {
				grid[r][c] ^= maskBit(m, r, c)
			}
		}
	}
}

// SelectMask renders every candidate mask onto a copy of unmasked
// (with PlaceFormatInfo for that mask), scores each with Penalty, and
// returns the (mask, grid) with the lowest score.
//
// unmasked is the grid after PlaceFunctionPatterns + PlaceData but
// BEFORE any mask or format-info placement. kinds is the
// corresponding Map.
func SelectMask(unmasked [][]byte, kinds *Map, s spec.Spec) (int, [][]byte) {
	best := -1
	var bestGrid [][]byte
	bestScore := 0
	for m := 0; m < MaskCount; m++ {
		candidate := cloneGrid(unmasked)
		ApplyMask(candidate, kinds, m)
		PlaceFormatInfo(candidate, s, m)
		score := Penalty(candidate)
		if best == -1 || score < bestScore {
			best = m
			bestScore = score
			bestGrid = candidate
		}
	}
	return best, bestGrid
}

func cloneGrid(src [][]byte) [][]byte {
	dst := make([][]byte, len(src))
	for r := range src {
		dst[r] = make([]byte, len(src[r]))
		copy(dst[r], src[r])
	}
	return dst
}
