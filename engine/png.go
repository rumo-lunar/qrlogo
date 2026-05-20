package engine

import (
	"image"
	"image/color"
	"image/png"
	"io"
)

// PNGOptions configure the PNG encoding of a Result.
type PNGOptions struct {
	// Scale is the side length in pixels of each QR module. Must be
	// > 0. Default 8 (a 61-module symbol with quiet zone 4 → 552 px
	// per side).
	Scale int

	// QuietZone is the width of the mandatory light border around
	// the symbol, in modules. Zero or negative means "use the
	// default" of 4 (the QR spec minimum for reliable scanning);
	// set explicitly to a positive value to override.
	QuietZone int
}

// EncodePNG writes the symbol as a 1-bit-per-pixel grayscale PNG.
//
// The output is square: (Symbol side + 2·QuietZone) × Scale pixels
// per side, in greyscale with dark modules = 0x00 and light modules
// (including the quiet zone) = 0xFF.
func (r *Result) EncodePNG(w io.Writer, opts PNGOptions) error {
	if opts.Scale <= 0 {
		opts.Scale = 8
	}
	if opts.QuietZone <= 0 {
		opts.QuietZone = 4
	}

	n := len(r.Symbol)
	side := (n + 2*opts.QuietZone) * opts.Scale

	img := image.NewGray(image.Rect(0, 0, side, side))
	for i := range img.Pix {
		img.Pix[i] = 0xFF
	}

	dark := color.Gray{Y: 0}
	for row := 0; row < n; row++ {
		for col := 0; col < n; col++ {
			if r.Symbol[row][col] != 1 {
				continue
			}
			x0 := (col + opts.QuietZone) * opts.Scale
			y0 := (row + opts.QuietZone) * opts.Scale
			for dy := 0; dy < opts.Scale; dy++ {
				for dx := 0; dx < opts.Scale; dx++ {
					img.SetGray(x0+dx, y0+dy, dark)
				}
			}
		}
	}

	return png.Encode(w, img)
}
