package qr

import "github.com/rumo-lunar/qrlogo/qr/spec"

// Symbol is the result of [Build]: the final masked NxN module grid
// plus the mask index that was picked by penalty scoring.
type Symbol struct {
	Grid [][]byte
	Mask int
}

// Build runs the full QR encoding pipeline for payload at spec s:
//
//  1. Frame payload (mode + char count + terminator + padding).
//  2. Split into RS blocks, compute EC per block, interleave.
//  3. Allocate the NxN grid and place function patterns.
//  4. Zig-zag place the data + EC bit stream.
//  5. Score all 8 masks (with the corresponding format-info bits)
//     by ISO/IEC 18004 §7.8.3 penalty and pick the lowest.
//
// Returns an error only if the payload exceeds s.MaxByteModePayload().
func Build(payload []byte, s spec.Spec) (*Symbol, error) {
	bits, err := BitStream(payload, s)
	if err != nil {
		return nil, err
	}

	n := s.Version.Size()
	grid := make([][]byte, n)
	for r := 0; r < n; r++ {
		grid[r] = make([]byte, n)
	}

	PlaceFunctionPatterns(grid, s.Version)
	kinds := NewMap(s.Version)
	PlaceData(grid, kinds, bits)

	mask, masked := SelectMask(grid, kinds, s)
	return &Symbol{Grid: masked, Mask: mask}, nil
}
