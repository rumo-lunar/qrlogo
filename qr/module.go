// Package qr builds the symbolic QR matrix used by the qrlogo
// constraint solver.
//
// The package is intentionally hardcoded for v1's contract: Version 11,
// EC level M, byte mode, mask 2. Generalising to other versions is a
// v2 concern.
package qr

// Kind classifies every module of the QR matrix by what placed it.
// Only KindData modules are available to carry symbolic data/EC bits;
// every other Kind has a value forced by the QR specification.
type Kind uint8

const (
	// KindData marks a module reserved for data or EC codeword bits.
	// These are the only positions where Phase 2 will place symbolic
	// linear forms.
	KindData Kind = iota

	// KindFinder marks one of the 7×7 finder-pattern modules at the
	// three corners of the symbol. Module value is fixed by the spec.
	KindFinder

	// KindSeparator marks the one-module-wide white ring on the
	// data-facing edges of each finder. Always 0 (white).
	KindSeparator

	// KindTiming marks the alternating black/white modules along
	// row 6 and column 6.
	KindTiming

	// KindAlignment marks the 5×5 alignment-pattern modules.
	KindAlignment

	// KindFormat marks the 30 format-information modules placed
	// around the three finders. Their bit values depend on EC level
	// and mask, but the *positions* are fixed.
	KindFormat

	// KindVersion marks the 36 version-information modules placed
	// near the top-right and bottom-left finders (only present on
	// versions ≥ 7).
	KindVersion

	// KindDark marks the single dark module at (4V+9, 8) — always 1.
	KindDark
)

// String renders Kind for debugging output.
func (k Kind) String() string {
	switch k {
	case KindData:
		return "Data"
	case KindFinder:
		return "Finder"
	case KindSeparator:
		return "Separator"
	case KindTiming:
		return "Timing"
	case KindAlignment:
		return "Alignment"
	case KindFormat:
		return "Format"
	case KindVersion:
		return "Version"
	case KindDark:
		return "Dark"
	}
	return "Unknown"
}

// v11Size is the side length in modules of a Version 11 QR symbol.
const v11Size = 61

// Map is the per-module classification of a QR symbol.
type Map struct {
	Size  int
	cells [][]Kind
}

// KindAt returns the Kind of the module at (row, col).
func (m *Map) KindAt(row, col int) Kind {
	return m.cells[row][col]
}

// NewV11Map constructs the function-pattern map for a Version 11 QR
// symbol. Every cell is initialised to KindData and then overlaid by
// each function-pattern category in an order that respects the QR
// spec's precedence rules:
//
//	finder > separator > alignment > timing > version > format > dark
//
// The post-condition checked by tests: exactly 3232 modules remain as
// KindData, matching V11's 404 codewords × 8 bits (with 0 remainder
// bits).
func NewV11Map() *Map {
	n := v11Size
	cells := make([][]Kind, n)
	for r := range cells {
		cells[r] = make([]Kind, n)
	}

	// --- 1. Finder patterns + separators at three corners ---
	// Top-left, top-right, bottom-left (no finder bottom-right).
	type finder struct{ r, c int } // top-left coordinate of the 7×7 block
	finders := []finder{
		{0, 0},
		{0, n - 7},
		{n - 7, 0},
	}
	for _, f := range finders {
		// 7×7 finder block.
		for dr := 0; dr < 7; dr++ {
			for dc := 0; dc < 7; dc++ {
				cells[f.r+dr][f.c+dc] = KindFinder
			}
		}
		// Separator: one-module ring on the data-facing edges of the
		// finder (the edges that touch the data region, not the symbol
		// outer edge).
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
	// V11 alignment-centre coordinates (ISO/IEC 18004 Annex E): 6, 30, 56.
	// Combinations overlapping a finder corner are excluded.
	centres := []int{6, 30, 56}
	for _, ar := range centres {
		for _, ac := range centres {
			if (ar == 6 && ac == 6) ||
				(ar == 6 && ac == 56) ||
				(ar == 56 && ac == 6) {
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
	// Row 6 and column 6, filling only cells still classified as KindData
	// (alignment and finder/separator take precedence).
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
	// Two 6×3 blocks: one above the bottom-left finder, one left of the
	// top-right finder.
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
	// 15-bit field placed in two strips around the finders.
	// Strip A (near top-left finder):
	//   row 8 cols 0..5, (8,7), (8,8), (7,8), rows 0..5 col 8.
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
	//   row 8 cols n-8..n-1, col 8 rows n-7..n-1.
	for c := n - 8; c < n; c++ {
		cells[8][c] = KindFormat
	}
	for r := n - 7; r < n; r++ {
		cells[r][8] = KindFormat
	}

	// --- 6. Dark module ---
	// Always 1, sits at (4V+9, 8). For V11 that is (53, 8).
	cells[4*11+9][8] = KindDark

	return &Map{Size: n, cells: cells}
}
