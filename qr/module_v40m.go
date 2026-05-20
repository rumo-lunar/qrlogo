package qr

// v40Size is the side length in modules of a Version 40 QR symbol.
const v40Size = 177

// NewV40Map constructs the function-pattern map for a Version 40 QR
// symbol. Every cell is initialised to KindData and then overlaid by
// each function-pattern category in an order that respects the QR
// spec's precedence rules:
//
//	finder > separator > alignment > timing > version > format > dark
//
// The post-condition: exactly 3706×8 = 29648 modules remain as
// KindData, matching V40's 3706 codewords × 8 bits (with 0 remainder
// bits).
func NewV40Map() *Map {
	n := v40Size
	cells := make([][]Kind, n)
	for r := range cells {
		cells[r] = make([]Kind, n)
	}

	// --- 1. Finder patterns + separators at three corners ---
	type finder struct{ r, c int }
	finders := []finder{
		{0, 0},
		{0, n - 7},
		{n - 7, 0},
	}
	for _, f := range finders {
		for dr := 0; dr < 7; dr++ {
			for dc := 0; dc < 7; dc++ {
				cells[f.r+dr][f.c+dc] = KindFinder
			}
		}
		switch {
		case f.r == 0 && f.c == 0: // top-left
			for c := 0; c < 8; c++ {
				cells[7][c] = KindSeparator
			}
			for r := 0; r < 7; r++ {
				cells[r][7] = KindSeparator
			}
		case f.r == 0 && f.c == n-7: // top-right
			for c := n - 8; c < n; c++ {
				cells[7][c] = KindSeparator
			}
			for r := 0; r < 7; r++ {
				cells[r][n-8] = KindSeparator
			}
		case f.r == n-7 && f.c == 0: // bottom-left
			for c := 0; c < 8; c++ {
				cells[n-8][c] = KindSeparator
			}
			for r := n - 7; r < n; r++ {
				cells[r][7] = KindSeparator
			}
		}
	}

	// --- 2. Alignment patterns ---
	// V40 alignment-centre coordinates (ISO/IEC 18004 dynamic formula):
	// 6, 30, 58, 86, 114, 142, 170.
	// Combinations overlapping a finder+separator corner are excluded:
	// (6,6), (6,170), (170,6).
	centresV40 := []int{6, 30, 58, 86, 114, 142, 170}
	for _, ar := range centresV40 {
		for _, ac := range centresV40 {
			if (ar == 6 && ac == 6) ||
				(ar == 6 && ac == 170) ||
				(ar == 170 && ac == 6) {
				continue
			}
			for dr := -2; dr <= 2; dr++ {
				for dc := -2; dc <= 2; dc++ {
					cells[ar+dr][ac+dc] = KindAlignment
				}
			}
		}
	}

	// --- 3. Timing patterns ---
	// Row 6 and column 6, filling only cells still classified as KindData.
	for c := 0; c < n; c++ {
		if cells[6][c] == KindData {
			cells[6][c] = KindTiming
		}
	}
	for r := 0; r < n; r++ {
		if cells[r][6] == KindData {
			cells[r][6] = KindTiming
		}
	}

	// --- 4. Version information (versions ≥ 7) ---
	// Block A: rows 0..5, columns n-11..n-9.
	for r := 0; r < 6; r++ {
		for c := n - 11; c <= n-9; c++ {
			cells[r][c] = KindVersion
		}
	}
	// Block B: rows n-11..n-9, columns 0..5.
	for r := n - 11; r <= n-9; r++ {
		for c := 0; c < 6; c++ {
			cells[r][c] = KindVersion
		}
	}

	// --- 5. Format information ---
	// Strip A (near top-left finder):
	for c := 0; c <= 5; c++ {
		cells[8][c] = KindFormat
	}
	cells[8][7] = KindFormat
	cells[8][8] = KindFormat
	cells[7][8] = KindFormat
	for r := 0; r <= 5; r++ {
		cells[r][8] = KindFormat
	}
	// Strip B (split between top-right and bottom-left finders):
	for c := n - 8; c < n; c++ {
		cells[8][c] = KindFormat
	}
	for r := n - 7; r < n; r++ {
		cells[r][8] = KindFormat
	}

	// --- 6. Dark module ---
	// Always 1, sits at (4V+9, 8). For V40 that is (169, 8).
	cells[4*40+9][8] = KindDark

	return &Map{Size: n, cells: cells}
}
