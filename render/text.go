package render

import (
	"image"
	"image/color"

	"golang.org/x/image/font/basicfont"
	"golang.org/x/image/font"
	"golang.org/x/image/math/fixed"
)

// RenderText rasterises text into a fresh w×h TargetMap using the
// builtin basicfont 7×13 face. Any pixel covered by a glyph becomes
// PixelBlack; every other cell stays PixelDontCare.
//
// The text baseline is positioned so the cap height is vertically
// centred and the first glyph is offset 1 cell from the left edge.
//
// Call ApplyHalo afterwards to surround the glyphs with a 1-cell
// PixelWhite halo so they remain legible against the noisy QR data
// modules.
//
// RenderText is intentionally agnostic about clipping: if the text
// runs past the right edge the glyphs are simply cut off. For a
// 177×177 V40 symbol the default 7×13 face fits roughly 25 chars per
// row.
func RenderText(text string, w, h int) *TargetMap {
	face := basicfont.Face7x13
	metrics := face.Metrics()
	originX := 1
	originY := (h+metrics.Ascent.Round()-metrics.Descent.Round())/2 - 1

	rgba := image.NewRGBA(image.Rect(0, 0, w, h))
	d := &font.Drawer{
		Dst:  rgba,
		Src:  image.NewUniform(color.Opaque),
		Face: face,
		Dot:  fixed.P(originX, originY),
	}
	d.DrawString(text)

	t := New(w, h)
	for r := 0; r < h; r++ {
		for c := 0; c < w; c++ {
			if _, _, _, a := rgba.At(c, r).RGBA(); a > 0 {
				t.Set(r, c, PixelBlack)
			}
		}
	}
	return t
}
