package render

import (
	"image"
	"image/color"

	"golang.org/x/image/font"
	"golang.org/x/image/font/basicfont"
	"golang.org/x/image/math/fixed"
)

// TextOptions configures how RenderText rasterises text.
type TextOptions struct {
	// Face is the font face used to draw glyphs. When nil, the
	// builtin basicfont.Face7x13 is used and the rendered bitmap is
	// nearest-neighbour upscaled to fill the target — useful for the
	// default "chunky pixel" look without any external font assets.
	//
	// When Face is non-nil it is rendered at its native size; the
	// caller is responsible for picking a size that fits w×h. No
	// upscaling is applied, so anti-aliased TTF/OTF glyphs stay
	// crisp.
	Face font.Face

	// AlphaThreshold is the minimum alpha value (in the 0..0xFFFF
	// range returned by color.Color.RGBA) at which a rasterised
	// pixel becomes PixelBlack. Lower values include more of the
	// anti-aliased glyph edge; higher values keep only the solid
	// core. Defaults to 0 — every opaque pixel counts.
	AlphaThreshold uint32
}

// RenderText rasterises text into a fresh w×h TargetMap. Any pixel
// covered by a glyph (above AlphaThreshold) becomes PixelBlack;
// every other cell stays PixelDontCare.
//
// With the default (basicfont) face the rendered bitmap is centred
// and nearest-neighbour upscaled by the largest integer factor that
// fits in w×h with a 1-cell margin. With a caller-supplied face the
// text is rendered at the face's native size and centred — no
// upscaling is performed.
//
// Call ApplyHalo afterwards to surround the glyphs with a 1-cell
// PixelWhite halo so they remain legible against the noisy QR data
// modules.
//
// RenderText never clips: if the requested w/h cannot hold the
// rendered glyphs any overflow on the right/bottom edges is silently
// dropped.
func RenderText(text string, w, h int, opts ...TextOptions) *TargetMap {
	t := New(w, h)
	if text == "" {
		return t
	}

	var o TextOptions
	if len(opts) > 0 {
		o = opts[0]
	}

	autoScale := o.Face == nil
	face := o.Face
	if face == nil {
		face = basicfont.Face7x13
	}

	metrics := face.Metrics()
	ascent := metrics.Ascent.Round()
	descent := metrics.Descent.Round()

	advance := font.MeasureString(face, text).Ceil()
	if advance <= 0 {
		return t
	}

	// Rasterise into a bitmap slightly larger than the advance width
	// to be safe against glyph overhang on either side.
	nativeW := advance + 2
	nativeH := ascent + descent + 2
	native := image.NewRGBA(image.Rect(0, 0, nativeW, nativeH))
	d := &font.Drawer{
		Dst:  native,
		Src:  image.NewUniform(color.Opaque),
		Face: face,
		Dot:  fixed.P(1, ascent+1),
	}
	d.DrawString(text)

	// Find the bounding box of pixels that pass the alpha threshold
	// so we centre what is visible rather than the advance width.
	minX, minY := nativeW, nativeH
	maxX, maxY := -1, -1
	for y := 0; y < nativeH; y++ {
		for x := 0; x < nativeW; x++ {
			if _, _, _, a := native.At(x, y).RGBA(); a > o.AlphaThreshold {
				if x < minX {
					minX = x
				}
				if x > maxX {
					maxX = x
				}
				if y < minY {
					minY = y
				}
				if y > maxY {
					maxY = y
				}
			}
		}
	}
	if maxX < 0 {
		// Nothing rendered (e.g. all chars missing from the font).
		return t
	}
	bbW := maxX - minX + 1
	bbH := maxY - minY + 1

	// Compute integer upscale factor. With a caller-supplied face we
	// stay at 1× to keep TTF/OTF anti-aliasing crisp. With basicfont
	// we expand by the largest factor that fits inside w×h minus a
	// 1-cell margin on every side.
	scale := 1
	if autoScale {
		const margin = 2 // 1 cell on each side
		scale = minInt((w-margin)/bbW, (h-margin)/bbH)
		if scale < 1 {
			scale = 1
		}
	}

	scaledW := bbW * scale
	scaledH := bbH * scale
	offsetX := (w - scaledW) / 2
	offsetY := (h - scaledH) / 2

	for r := 0; r < scaledH; r++ {
		dstR := offsetY + r
		if dstR < 0 || dstR >= h {
			continue
		}
		srcY := minY + r/scale
		for c := 0; c < scaledW; c++ {
			dstC := offsetX + c
			if dstC < 0 || dstC >= w {
				continue
			}
			srcX := minX + c/scale
			if _, _, _, a := native.At(srcX, srcY).RGBA(); a > o.AlphaThreshold {
				t.Set(dstR, dstC, PixelBlack)
			}
		}
	}
	return t
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
