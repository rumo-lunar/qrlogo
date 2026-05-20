package render

// ApplyHalo converts the 1-cell ring of PixelDontCare around every
// PixelBlack into PixelWhite, in place.
//
// Intuition: the QR symbol is noisy random modules. Without a halo,
// the solver is free to paint a dark module immediately adjacent to
// an intended dark logo pixel, which causes the logo to blur into
// the surrounding pattern. Forcing the immediate neighbours to be
// light gives the logo a guaranteed 1-cell white outline against
// which it reads clearly.
//
// Cells that are already PixelBlack or PixelWhite are left untouched
// — only existing PixelDontCare cells can be promoted. Halos do not
// extend beyond the grid edges.
//
// The mutation is computed from a snapshot of the input so that a
// halo pixel created in this pass does not seed further halo
// expansion (i.e. halos are strictly 1 cell wide).
func ApplyHalo(t *TargetMap) {
	if t == nil {
		return
	}
	snapshot := make([]PixelState, len(t.Pixels))
	copy(snapshot, t.Pixels)

	at := func(r, c int) PixelState {
		return snapshot[r*t.W+c]
	}

	for r := 0; r < t.H; r++ {
		for c := 0; c < t.W; c++ {
			if at(r, c) != PixelDontCare {
				continue
			}
			if hasBlackNeighbour(t, at, r, c) {
				t.Set(r, c, PixelWhite)
			}
		}
	}
}

func hasBlackNeighbour(t *TargetMap, at func(int, int) PixelState, r, c int) bool {
	for dr := -1; dr <= 1; dr++ {
		for dc := -1; dc <= 1; dc++ {
			if dr == 0 && dc == 0 {
				continue
			}
			nr, nc := r+dr, c+dc
			if !t.Inside(nr, nc) {
				continue
			}
			if at(nr, nc) == PixelBlack {
				return true
			}
		}
	}
	return false
}
