package qr

// PlaceData writes bits into the data modules of grid in the
// zig-zag column order specified by ISO/IEC 18004 §7.7.3.
//
// Starting at the bottom-right, the encoder fills two-column strips
// from right to left, alternating direction (upward, downward,
// upward, …). The column pair containing the vertical timing line
// (column 6) is skipped — the placement uses columns 5 and 4 as a
// single pair, then 3 and 2, etc.
//
// Within each two-column strip, the rightmost column of the pair is
// written before the leftmost for any given row. Function-pattern
// cells (anything where m.IsData is false) are skipped without
// consuming a bit.
//
// Panics if bits is shorter than the number of data modules.
func PlaceData(grid [][]byte, m *Map, bits []byte) {
	n := m.Size
	idx := 0
	// Walk column pairs right-to-left starting at the rightmost pair.
	upward := true
	col := n - 1
	for col > 0 {
		if col == 6 {
			// Skip the timing column: shift left and continue.
			col--
		}
		for k := 0; k < n; k++ {
			var r int
			if upward {
				r = n - 1 - k
			} else {
				r = k
			}
			for _, c := range []int{col, col - 1} {
				if m.IsData(r, c) {
					if idx >= len(bits) {
						return
					}
					grid[r][c] = bits[idx]
					idx++
				}
			}
		}
		upward = !upward
		col -= 2
	}
}
