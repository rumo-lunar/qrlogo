package qr

// FormatV40MMask2 is the 15-bit format string for EC level M and
// mask pattern 2. This is identical to FormatV11MMask2 because format
// information depends only on EC level and mask, not on the QR version.
//
// Value: 101111001111100 (same derivation as FormatV11MMask2).
const FormatV40MMask2 uint16 = 0b101111001111100

// VersionV40 is the 18-bit version-information string for Version 40:
//
//  1. Version number 40 = "101000" (6 bits)
//  2. BCH(18, 6) by G(x) = x^12 + x^11 + x^10 + x^9 + x^8 + x^5 + x^2 + 1
//     remainder = 110001101001 (12 bits)
//  3. Concatenated: 101000 110001101001
//
// MSB-first, so bit 17 (=1) is the leftmost bit.
const VersionV40 uint32 = 0b101000110001101001

// FunctionBitsV40M returns the 177×177 grid of concrete bit values
// (0 = light, 1 = dark) for every non-data cell of a V40 QR symbol
// with EC level M and mask 2. KindData cells are zero-filled
// placeholders; callers combine this grid with the symbolic data
// grid produced by ApplyMask2V40M(PlaceCodewordsV40M(…)) to obtain
// the full rendered symbol.
//
// Cells set here:
//
//   - finder patterns at the three corners,
//   - separators (left as 0, the spec value),
//   - timing patterns on row 6 and column 6,
//   - 46 alignment patterns at the non-excluded combinations of
//     {6,30,58,86,114,142,170}×{6,30,58,86,114,142,170},
//   - dark module at (169, 8),
//   - 30 format-info modules with FormatV40MMask2,
//   - 36 version-info modules with VersionV40.
func FunctionBitsV40M() [][]byte {
	n := v40Size
	g := make([][]byte, n)
	for r := range g {
		g[r] = make([]byte, n)
	}

	placeFinders(g, n)
	placeTimingV40(g, n)
	placeAlignmentV40(g)
	placeDarkModuleV40(g)
	placeFormatBits(g, n, FormatV40MMask2)
	placeVersionBits(g, n, VersionV40)
	// Separators stay 0 — they are always light by spec.
	return g
}

// placeTimingV40 fills the timing modules at row 6 and column 6 for
// V40. It uses NewV40Map to respect alignment-pattern precedence.
func placeTimingV40(g [][]byte, n int) {
	m := NewV40Map()
	for r := 0; r < n; r++ {
		for c := 0; c < n; c++ {
			if m.KindAt(r, c) != KindTiming {
				continue
			}
			var darkIfEven int
			if r == 6 {
				darkIfEven = c
			} else {
				darkIfEven = r
			}
			if darkIfEven%2 == 0 {
				g[r][c] = 1
			}
		}
	}
}

// placeAlignmentV40 writes the 46 alignment patterns for V40.
func placeAlignmentV40(g [][]byte) {
	centres := [7]int{6, 30, 58, 86, 114, 142, 170}
	last := centres[len(centres)-1]
	for _, ar := range centres {
		for _, ac := range centres {
			if (ar == centres[0] && ac == centres[0]) ||
				(ar == centres[0] && ac == last) ||
				(ar == last && ac == centres[0]) {
				continue
			}
			for dr := -2; dr <= 2; dr++ {
				for dc := -2; dc <= 2; dc++ {
					g[ar+dr][ac+dc] = alignmentPattern[dr+2][dc+2]
				}
			}
		}
	}
}

func placeDarkModuleV40(g [][]byte) {
	g[4*40+9][8] = 1 // (169, 8) for V40
}
