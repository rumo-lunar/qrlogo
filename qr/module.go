// Package qr builds the symbolic QR matrix used by the qrlogo
// constraint solver.
//
// The package is hardcoded for Version 40, EC level M, byte mode,
// mask 2. Generalising to other versions or EC levels is out of
// scope.
package qr

// Kind classifies every module of the QR matrix by what placed it.
// Only KindData modules are available to carry symbolic data/EC bits;
// every other Kind has a value forced by the QR specification.
type Kind uint8

const (
	// KindData marks a module reserved for data or EC codeword bits.
	// These are the only positions where the engine will place symbolic
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

// Size is the side length in modules of a Version 40 QR symbol.
const Size = 177

// Map is the per-module classification of a QR symbol.
type Map struct {
	Size  int
	cells [][]Kind
}

// KindAt returns the Kind of the module at (row, col).
func (m *Map) KindAt(row, col int) Kind {
	return m.cells[row][col]
}

// NewMap constructs the function-pattern map for a Version 40 QR
// symbol. Every cell is initialised to KindData and then overlaid by
// each function-pattern category in an order that respects the QR
// spec's precedence rules:
//
//	finder > separator > alignment > timing > version > format > dark
//
// The post-condition: exactly 3706×8 = 29648 modules remain as
// KindData, matching V40's 3706 codewords × 8 bits (with 0 remainder
// bits).
func NewMap() *Map {
	n := Size
	cells := make([][]Kind, n)
	for r := range cells {
		cells[r] = make([]Kind, n)
	}

	// --- 1. Finder patterns + separators at three corners ---
	for _, f := range finderOrigins {
		fr, fc := f[0], f[1]
		for dr := 0; dr < 7; dr++ {
			for dc := 0; dc < 7; dc++ {
				cells[fr+dr][fc+dc] = KindFinder
			}
		}
		switch {
		case fr == 0 && fc == 0: // top-left
			for c := 0; c < 8; c++ {
				cells[7][c] = KindSeparator
			}
			for r := 0; r < 7; r++ {
				cells[r][7] = KindSeparator
			}
		case fr == 0 && fc == n-7: // top-right
			for c := n - 8; c < n; c++ {
				cells[7][c] = KindSeparator
			}
			for r := 0; r < 7; r++ {
				cells[r][n-8] = KindSeparator
			}
		case fr == n-7 && fc == 0: // bottom-left
			for c := 0; c < 8; c++ {
				cells[n-8][c] = KindSeparator
			}
			for r := n - 7; r < n; r++ {
				cells[r][7] = KindSeparator
			}
		}
	}

	// --- 2. Alignment patterns ---
	forEachAlignment(func(ar, ac int) {
		for dr := -2; dr <= 2; dr++ {
			for dc := -2; dc <= 2; dc++ {
				cells[ar+dr][ac+dc] = KindAlignment
			}
		}
	})

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
