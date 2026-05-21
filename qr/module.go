package qr

import "github.com/rumo-lunar/qrlogo/qr/spec"

// Kind classifies the role of a single module in a QR symbol.
//
// The dark/light value of every non-Data module is fixed by the spec
// (for Finder, Separator, Timing, Alignment and the Dark module) or
// by the chosen (EC, mask) pair (for Format) or by the Version (for
// Version). Only Data modules carry the encoded codeword bits.
type Kind uint8

const (
	KindData Kind = iota
	KindFinder
	KindSeparator
	KindTiming
	KindAlignment
	KindFormat
	KindVersion
	KindDark
)

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
	return "?"
}

// Map labels every cell of a Spec.Version-sized grid with its Kind.
type Map struct {
	Size  int
	Cells [][]Kind
}

// NewMap builds the function-pattern map for version v. Assignment
// priority is: Dark > Version > Format > Alignment > Timing >
// Separator > Finder > Data. Higher entries override lower ones when
// regions overlap.
func NewMap(v spec.Version) *Map {
	n := v.Size()
	cells := make([][]Kind, n)
	for r := 0; r < n; r++ {
		cells[r] = make([]Kind, n)
	}

	// 1. Finders: 7×7 at each of the three corners.
	for _, o := range v.FinderOrigins() {
		for dr := 0; dr < 7; dr++ {
			for dc := 0; dc < 7; dc++ {
				cells[o[0]+dr][o[1]+dc] = KindFinder
			}
		}
	}

	// 2. Separators: 1-cell light border around each finder. Only
	//    write inside the grid.
	for _, o := range v.FinderOrigins() {
		r0, c0 := o[0], o[1]
		for k := -1; k <= 7; k++ {
			setIf(cells, r0-1, c0+k, KindSeparator)
			setIf(cells, r0+7, c0+k, KindSeparator)
			setIf(cells, r0+k, c0-1, KindSeparator)
			setIf(cells, r0+k, c0+7, KindSeparator)
		}
	}

	// 3. Timing patterns: row 6 and col 6 across the inner region.
	for k := 8; k < n-8; k++ {
		cells[6][k] = KindTiming
		cells[k][6] = KindTiming
	}

	// 4. Alignment patterns: 5×5 at every centre not overlapping a finder.
	v.ForEachAlignment(func(ar, ac int) {
		for dr := -2; dr <= 2; dr++ {
			for dc := -2; dc <= 2; dc++ {
				cells[ar+dr][ac+dc] = KindAlignment
			}
		}
	})

	// 5. Format info reserved cells (placed before Dark so Dark wins
	//    at (4V+9, 8)).
	for k := 0; k <= 8; k++ {
		setIfFormat(cells, 8, k)
		setIfFormat(cells, k, 8)
	}
	for k := 0; k < 8; k++ {
		setIfFormat(cells, 8, n-1-k)
		setIfFormat(cells, n-1-k, 8)
	}

	// 6. Version info (V ≥ 7): 6×3 block above bottom-left finder and
	//    3×6 block left of top-right finder.
	if v.HasVersionInfo() {
		for i := 0; i < 18; i++ {
			r := n - 11 + (i % 3)
			c := i / 3
			cells[r][c] = KindVersion
			cells[c][r] = KindVersion
		}
	}

	// 7. Dark module: always 1, sits at (4V+9, 8).
	dr, dc := v.DarkModule()
	cells[dr][dc] = KindDark

	return &Map{Size: n, Cells: cells}
}

// KindAt returns the Kind at (r, c).
func (m *Map) KindAt(r, c int) Kind { return m.Cells[r][c] }

// IsData reports whether (r, c) is a data module.
func (m *Map) IsData(r, c int) bool { return m.Cells[r][c] == KindData }

// setIf writes k at (r, c) iff the coordinates are inside the grid
// and the current cell is still Data (priority: don't overwrite a
// stronger Kind that was already assigned).
func setIf(cells [][]Kind, r, c int, k Kind) {
	n := len(cells)
	if r < 0 || r >= n || c < 0 || c >= n {
		return
	}
	if cells[r][c] == KindData {
		cells[r][c] = k
	}
}

// setIfFormat sets cells[r][c] = KindFormat unless something stronger
// already lives there. Format reservation may legitimately overlap
// the separators around finders (cells 0..7 of row 8 and col 8 of
// the top-left finder, etc.); in those cases the format-info bit
// wins because format-info will actually be written, while separator
// cells are just structural light borders that get rendered as 0
// anyway. We still tag them KindFormat so placement skips them.
func setIfFormat(cells [][]Kind, r, c int) {
	n := len(cells)
	if r < 0 || r >= n || c < 0 || c >= n {
		return
	}
	switch cells[r][c] {
	case KindFinder, KindAlignment, KindTiming:
		// Don't clobber the actual function patterns the format-info
		// region sits next to; the format positions intentionally
		// avoid those cells.
		return
	}
	cells[r][c] = KindFormat
}
