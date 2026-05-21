package render

import "fmt"

// TargetMap is a row-major W×H grid of PixelState values.
//
// It is the lingua franca between the /render package (which produces
// visual constraints) and the /engine package (which turns those
// constraints into rows of the GF(2) linear system solved by /bitset).
type TargetMap struct {
	W, H   int
	Pixels []PixelState
}

// New allocates a TargetMap of the requested dimensions. All cells
// start as PixelDontCare (the zero value).
func New(w, h int) *TargetMap {
	if w <= 0 || h <= 0 {
		panic(fmt.Sprintf("render: invalid TargetMap size %dx%d", w, h))
	}
	return &TargetMap{
		W:      w,
		H:      h,
		Pixels: make([]PixelState, w*h),
	}
}

// At returns the pixel at (row, col). Panics on out-of-bounds access.
func (t *TargetMap) At(row, col int) PixelState {
	return t.Pixels[t.index(row, col)]
}

// Set writes p at (row, col). Panics on out-of-bounds access.
func (t *TargetMap) Set(row, col int, p PixelState) {
	t.Pixels[t.index(row, col)] = p
}

// Inside reports whether (row, col) lies within the grid.
func (t *TargetMap) Inside(row, col int) bool {
	return row >= 0 && row < t.H && col >= 0 && col < t.W
}

// Counts returns the number of Black, White and DontCare cells.
func (t *TargetMap) Counts() (black, white, dontCare int) {
	for _, p := range t.Pixels {
		switch p {
		case PixelBlack:
			black++
		case PixelWhite:
			white++
		case PixelDontCare:
			dontCare++
		}
	}
	return
}

// String returns a multi-line ASCII rendering. Useful in test
// failures and for hand-authored expected fixtures.
func (t *TargetMap) String() string {
	out := make([]byte, 0, (t.W+1)*t.H)
	for r := 0; r < t.H; r++ {
		for c := 0; c < t.W; c++ {
			out = append(out, t.At(r, c).String()[0])
		}
		out = append(out, '\n')
	}
	return string(out)
}

// ForEachConstraint invokes fn for every cell whose state is
// PixelBlack or PixelWhite, supplying the cell coordinates and the
// required bit (1 for Black, 0 for White). DontCare cells are
// silently skipped.
//
// This is the "tell, don't ask" entry point the engine uses to build
// its constraint system: the engine tells the TargetMap to deliver
// every constraint instead of polling every pixel and discarding the
// DontCare ones itself.
func (t *TargetMap) ForEachConstraint(fn func(row, col int, wantBit byte)) {
	for r := 0; r < t.H; r++ {
		base := r * t.W
		for c := 0; c < t.W; c++ {
			switch t.Pixels[base+c] {
			case PixelBlack:
				fn(r, c, 1)
			case PixelWhite:
				fn(r, c, 0)
			}
		}
	}
}

func (t *TargetMap) index(row, col int) int {
	if !t.Inside(row, col) {
		panic(fmt.Sprintf("render: (%d,%d) out of bounds for %dx%d map",
			row, col, t.H, t.W))
	}
	return row*t.W + col
}
