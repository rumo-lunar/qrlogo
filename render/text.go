package render

import (
	"image"
	"image/color"

	"golang.org/x/image/font"
	"golang.org/x/image/font/basicfont"
	"golang.org/x/image/math/fixed"
)

// TextOptions configure how text is rasterised into a TargetMap.
type TextOptions struct {
	// Face is the font face used for rasterisation. If nil,
	// basicfont.Face7x13 is used (7px wide, 13px tall glyphs).
	Face font.Face

	// OriginX, OriginY place the baseline of the first glyph (in
	// pixel coordinates of the resulting grid). If OriginY is zero
	// the renderer picks a baseline that vertically centres the
	// glyph cap height in the grid; if OriginX is zero it picks
	// a value that left-aligns the text with a 1-cell margin.
	OriginX, OriginY int
}

// RenderText rasterises text into a fresh w×h TargetMap. Any pixel
// covered by a glyph becomes PixelBlack; every other cell stays
// PixelDontCare.
//
// Call ApplyHalo afterwards to surround the glyphs with a 1-cell
// PixelWhite halo so they remain legible against the noisy QR data
// modules.
//
// RenderText is intentionally agnostic about clipping: if the text
// runs past the right edge the glyphs are simply cut off. Callers
// are expected to pick a font face that fits — for a 61×61 V11
// symbol the default 7×13 face fits roughly 8 characters per row.
func RenderText(text string, w, h int, opts TextOptions) *TargetMap {
	if opts.Face == nil {
		opts.Face = basicfont.Face7x13
	}

	originX, originY := opts.OriginX, opts.OriginY
	if originX == 0 {
		originX = 1
	}
	if originY == 0 {
		// Centre the cap height roughly in the grid.
		metrics := opts.Face.Metrics()
		ascent := metrics.Ascent.Round()
		descent := metrics.Descent.Round()
		originY = (h+ascent-descent)/2 - 1
	}

	rgba := image.NewRGBA(image.Rect(0, 0, w, h))
	// Background stays transparent; only drawn glyph pixels matter.
	d := &font.Drawer{
		Dst:  rgba,
		Src:  image.NewUniform(color.Opaque),
		Face: opts.Face,
		Dot:  fixed.P(originX, originY),
	}
	d.DrawString(text)

	t := New(w, h)
	for r := 0; r < h; r++ {
		for c := 0; c < w; c++ {
			_, _, _, a := rgba.At(c, r).RGBA()
			if a > 0 {
				t.Set(r, c, PixelBlack)
			}
		}
	}
	return t
}
