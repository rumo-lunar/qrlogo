package qr

// Function-pattern bit values for a V11 QR symbol with EC level M
// and mask 2 applied. These are the concrete bits forced by the QR
// spec at every non-data module; the engine reads them to decide
// whether an image pixel that lands on a function-pattern cell is
// already satisfied or unattainable.
//
// All constants and patterns in this file are derived from
// ISO/IEC 18004 (annexes for finder/alignment patterns, §7.9 for
// format info, §7.10 for version info) and reproduced here as
// literals so the values are auditable from a single page.

// FormatV11MMask2 is the 15-bit format string for EC level M and
// mask pattern 2:
//
//   1. EC-level bits  M  = "00"      (ISO Table 12)
//   2. Mask number    2  = "010"     (ISO Table 23)
//   3. 5-bit field    = "00010"
//   4. BCH(15, 5)     append 10 zero bits, divide by
//      G(x) = x^10 + x^8 + x^5 + x^4 + x^2 + x + 1, take remainder
//      → 0001010011 01110 (15 bits)
//   5. XOR with mask 0x5412 = 101010000010010
//      → 101111001111100
//
// MSB-first, so bit 14 (=1) is the leftmost bit.
const FormatV11MMask2 uint16 = 0b101111001111100

// VersionV11 is the 18-bit version-information string for Version 11:
//
//   1. Version number 11 = "001011" (6 bits)
//   2. BCH(18, 6) by G(x) = x^12 + x^11 + x^10 + x^9 + x^8 + x^5 + x^2 + 1
//      remainder = 101111110110 (12 bits)
//   3. Concatenated: 001011 101111110110
//
// MSB-first, so bit 17 (=0) is the leftmost bit.
const VersionV11 uint32 = 0b001011101111110110

// finderPattern is the 7×7 dark/light pattern of the QR finder.
var finderPattern = [7][7]byte{
	{1, 1, 1, 1, 1, 1, 1},
	{1, 0, 0, 0, 0, 0, 1},
	{1, 0, 1, 1, 1, 0, 1},
	{1, 0, 1, 1, 1, 0, 1},
	{1, 0, 1, 1, 1, 0, 1},
	{1, 0, 0, 0, 0, 0, 1},
	{1, 1, 1, 1, 1, 1, 1},
}

// alignmentPattern is the 5×5 alignment pattern.
var alignmentPattern = [5][5]byte{
	{1, 1, 1, 1, 1},
	{1, 0, 0, 0, 1},
	{1, 0, 1, 0, 1},
	{1, 0, 0, 0, 1},
	{1, 1, 1, 1, 1},
}

// FunctionBitsV11M returns the 61×61 grid of concrete bit values
// (0 = light, 1 = dark) for every non-data cell of a V11 QR symbol
// with EC level M and mask 2. KindData cells are zero-filled
// placeholders; callers combine this grid with the symbolic data
// grid produced by ApplyMask2(PlaceCodewords(…)) to obtain the full
// rendered symbol once /bitset has solved for the free variables.
//
// Cells set here:
//
//   - finder patterns at the three corners,
//   - separators (left as 0, the spec value),
//   - timing patterns on row 6 and column 6,
//   - 6 alignment patterns at centres {6,30,56}×{6,30,56} minus
//     finder overlaps,
//   - dark module at (53, 8),
//   - 30 format-info modules with FormatV11MMask2,
//   - 36 version-info modules with VersionV11.
func FunctionBitsV11M() [][]byte {
	n := v11Size
	g := make([][]byte, n)
	for r := range g {
		g[r] = make([]byte, n)
	}

	placeFinders(g, n)
	placeTiming(g, n)
	placeAlignment(g, n)
	placeDarkModule(g)
	placeFormatBits(g, n, FormatV11MMask2)
	placeVersionBits(g, n, VersionV11)
	// Separators stay 0 — they are always light by spec.
	return g
}

func placeFinders(g [][]byte, n int) {
	corners := [3][2]int{
		{0, 0},
		{0, n - 7},
		{n - 7, 0},
	}
	for _, p := range corners {
		for dr := 0; dr < 7; dr++ {
			for dc := 0; dc < 7; dc++ {
				g[p[0]+dr][p[1]+dc] = finderPattern[dr][dc]
			}
		}
	}
}

// placeTiming fills the timing modules at row 6 and column 6. The
// pattern alternates starting from dark; only cells whose Kind is
// KindTiming (per NewV11Map) are written, which correctly handles
// the gaps where alignment patterns interrupt the timing strips.
//
// Formula: a timing module is dark iff its varying coordinate is
// even. Horizontal strip (row 6) uses column index; vertical strip
// (col 6) uses row index.
func placeTiming(g [][]byte, n int) {
	m := NewV11Map()
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

func placeAlignment(g [][]byte, n int) {
	// V11 alignment centres from ISO Annex E.
	centres := [3]int{6, 30, 56}
	for _, ar := range centres {
		for _, ac := range centres {
			if (ar == 6 && ac == 6) ||
				(ar == 6 && ac == 56) ||
				(ar == 56 && ac == 6) {
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

func placeDarkModule(g [][]byte) {
	g[4*11+9][8] = 1 // (53, 8) for V11
}

// placeFormatBits writes the 15-bit format string at the two
// canonical locations defined in ISO §7.9.1. Within each location
// the bit order runs from MSB (bit 14) to LSB (bit 0).
func placeFormatBits(g [][]byte, n int, format uint16) {
	bits := make([]byte, 15)
	for i := 0; i < 15; i++ {
		bits[i] = byte((format >> uint(14-i)) & 1)
	}

	// Location 1: around the top-left finder.
	loc1 := [15][2]int{
		{8, 0}, {8, 1}, {8, 2}, {8, 3}, {8, 4}, {8, 5}, // bits 14..9
		{8, 7}, {8, 8}, {7, 8}, // bits 8..6 (col 6 is timing, skip)
		{5, 8}, {4, 8}, {3, 8}, {2, 8}, {1, 8}, {0, 8}, // bits 5..0
	}
	for i, p := range loc1 {
		g[p[0]][p[1]] = bits[i]
	}

	// Location 2: split between top-right and bottom-left finders.
	// Bits 14..8 along col 8 of the bottom-left (top-down from bottom),
	// bits 7..0 along row 8 of the top-right (left-to-right).
	loc2 := [15][2]int{
		{n - 1, 8}, {n - 2, 8}, {n - 3, 8}, {n - 4, 8}, {n - 5, 8}, {n - 6, 8}, {n - 7, 8}, // bits 14..8
		{8, n - 8}, {8, n - 7}, {8, n - 6}, {8, n - 5}, {8, n - 4}, {8, n - 3}, {8, n - 2}, {8, n - 1}, // bits 7..0
	}
	for i, p := range loc2 {
		g[p[0]][p[1]] = bits[i]
	}
}

// placeVersionBits writes the 18-bit version-info string at the
// two canonical 6×3 blocks defined in ISO §7.10. The bit numbering
// in each block is identical:
//
//   - block at (0..5, n-11..n-9): cell (r, n-11+j) holds bit r·3+j
//   - block at (n-11..n-9, 0..5): cell (n-11+i, c) holds bit c·3+i
//
// Both blocks carry the same 18 bits, the layouts mirror each other.
func placeVersionBits(g [][]byte, n int, version uint32) {
	bit := func(i int) byte { return byte((version >> uint(i)) & 1) }

	// Block A: top-right area, 6 rows × 3 cols.
	for r := 0; r < 6; r++ {
		for j := 0; j < 3; j++ {
			g[r][n-11+j] = bit(r*3 + j)
		}
	}
	// Block B: bottom-left area, 3 rows × 6 cols.
	for i := 0; i < 3; i++ {
		for c := 0; c < 6; c++ {
			g[n-11+i][c] = bit(c*3 + i)
		}
	}
}
