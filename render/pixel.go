// Package render builds visual target maps for the QArt engine.
//
// A target map is a grid of PixelState values the engine treats as
// constraints when solving for the free QR padding bits:
//
//   - PixelBlack    — the cell MUST be a dark QR module
//   - PixelWhite    — the cell MUST be a light QR module
//   - PixelDontCare — the solver may pick either; no constraint
//
// The two input methods (RenderText, FromImage) only ever produce
// PixelBlack and PixelDontCare pixels. Call ApplyHalo to convert the
// 1-cell ring of DontCare around every Black into PixelWhite, which
// is what makes the embedded logo visually stand out against the
// noisy QR background.
package render

// PixelState is the three-valued logic used by the target map.
type PixelState uint8

const (
	// PixelDontCare is the zero value: solver is free to choose.
	PixelDontCare PixelState = iota
	// PixelWhite forces the QR module at this cell to be light.
	PixelWhite
	// PixelBlack forces the QR module at this cell to be dark.
	PixelBlack
)

// String returns a one-character label useful in tests and debug dumps.
//
//   '.' = don't care, ' ' = white, '#' = black, '?' = unknown
func (p PixelState) String() string {
	switch p {
	case PixelDontCare:
		return "."
	case PixelWhite:
		return " "
	case PixelBlack:
		return "#"
	default:
		return "?"
	}
}
