package qr

import "github.com/rumo-lunar/qrlogo/qr/spec"

// finderBit returns the dark/light value of the 7×7 finder pattern
// at the (dr, dc) offset within the finder (0 ≤ dr, dc < 7).
//
// The pattern is a 3×3 dark square in a 5×5 light square in a 7×7
// dark frame:
//
//	1 1 1 1 1 1 1
//	1 0 0 0 0 0 1
//	1 0 1 1 1 0 1
//	1 0 1 1 1 0 1
//	1 0 1 1 1 0 1
//	1 0 0 0 0 0 1
//	1 1 1 1 1 1 1
func finderBit(dr, dc int) byte {
	if dr == 0 || dr == 6 || dc == 0 || dc == 6 {
		return 1
	}
	if dr >= 2 && dr <= 4 && dc >= 2 && dc <= 4 {
		return 1
	}
	return 0
}

// alignmentBit returns the dark/light value of the 5×5 alignment
// pattern at the (dr, dc) offset within the pattern (−2 ≤ dr, dc ≤ 2):
//
//	1 1 1 1 1
//	1 0 0 0 1
//	1 0 1 0 1
//	1 0 0 0 1
//	1 1 1 1 1
func alignmentBit(dr, dc int) byte {
	if dr == 0 && dc == 0 {
		return 1
	}
	if dr == -2 || dr == 2 || dc == -2 || dc == 2 {
		return 1
	}
	return 0
}

// PlaceFunctionPatterns writes the spec-fixed function-pattern bits
// (finders, separators, timing, alignment, dark module, version info
// if V ≥ 7) onto grid. Format-info cells are left as zero — they are
// filled per-mask by PlaceFormatInfo.
//
// grid must be a v.Size() × v.Size() byte matrix preallocated by the
// caller. Cells outside the function regions are not touched.
func PlaceFunctionPatterns(grid [][]byte, v spec.Version) {
	n := v.Size()

	// Finders.
	for _, o := range v.FinderOrigins() {
		for dr := 0; dr < 7; dr++ {
			for dc := 0; dc < 7; dc++ {
				grid[o[0]+dr][o[1]+dc] = finderBit(dr, dc)
			}
		}
	}
	// Separators stay zero (light). They are KindSeparator in the
	// Map but bit value 0, which is the zero-initialised state.

	// Timing patterns.
	for k := 8; k < n-8; k++ {
		bit := byte(1 - (k % 2)) // even cols/rows are dark
		grid[6][k] = bit
		grid[k][6] = bit
	}

	// Alignment patterns.
	v.ForEachAlignment(func(ar, ac int) {
		for dr := -2; dr <= 2; dr++ {
			for dc := -2; dc <= 2; dc++ {
				grid[ar+dr][ac+dc] = alignmentBit(dr, dc)
			}
		}
	})

	// Dark module.
	dr, dc := v.DarkModule()
	grid[dr][dc] = 1

	// Version info (V ≥ 7), 18 bits MSB-first into two 3×6 blocks.
	if v.HasVersionInfo() {
		vi := v.VersionInfo()
		for i := 0; i < 18; i++ {
			bit := byte((vi >> uint(17-i)) & 1)
			r := n - 11 + (i % 3)
			c := i / 3
			grid[r][c] = bit
			grid[c][r] = bit
		}
	}
}

// PlaceFormatInfo writes the 15-bit format-information string for
// (s, mask) into grid, in both of the two prescribed copies
// (ISO/IEC 18004 §7.9.1). Bit 14 of FormatInfo is placed first.
func PlaceFormatInfo(grid [][]byte, s spec.Spec, mask int) {
	n := s.Version.Size()
	fi := s.FormatInfo(mask)
	bit := func(i int) byte { return byte((fi >> uint(14-i)) & 1) }

	// Copy 1 — around the top-left finder.
	// Bits 0..5 along row 8, cols 0..5.
	for i := 0; i < 6; i++ {
		grid[8][i] = bit(i)
	}
	grid[8][7] = bit(6)
	grid[8][8] = bit(7)
	grid[7][8] = bit(8)
	// Bits 9..14 along col 8, rows 5..0.
	for i := 9; i < 15; i++ {
		grid[14-i][8] = bit(i)
	}

	// Copy 2 — split across the bottom-left and top-right corners.
	// Bits 0..6 along col 8, rows n-1..n-7.
	for i := 0; i < 7; i++ {
		grid[n-1-i][8] = bit(i)
	}
	// Bits 7..14 along row 8, cols n-8..n-1.
	for i := 7; i < 15; i++ {
		grid[8][n-15+i] = bit(i)
	}
}
