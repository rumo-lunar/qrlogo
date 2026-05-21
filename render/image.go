package render

import (
	"image"

	"golang.org/x/image/draw"
)

// ImageOptions configure FromImage's thresholding behaviour.
type ImageOptions struct {
	// Threshold is the luminance cutoff in the range [0, 65535].
	// Pixels with luminance strictly less than Threshold become
	// PixelBlack. Pixels at or above Threshold stay PixelDontCare,
	// leaving the QR solver free to pick either.
	//
	// Zero means use 0x8000 (mid-grey).
	Threshold uint32

	// IgnoreTransparent, when true, leaves source pixels with zero
	// alpha as PixelDontCare regardless of any other rule. This is
	// what you want for logos with a transparent background.
	IgnoreTransparent bool
}

// FromImage downsamples src to w×h pixels and thresholds each pixel
// into a PixelState. The result has only PixelBlack and PixelDontCare.
//
// Call ApplyHalo afterwards to add a 1-cell PixelWhite halo around
// the dark features for visual contrast.
//
// Aspect ratio is NOT preserved — the entire source image is squeezed
// into the target dimensions. Pre-crop or pad the input yourself if
// you need square or other ratios.
func FromImage(src image.Image, w, h int, opts ImageOptions) *TargetMap {
	if opts.Threshold == 0 {
		opts.Threshold = 0x8000
	}

	rgba := image.NewRGBA(image.Rect(0, 0, w, h))
	draw.CatmullRom.Scale(rgba, rgba.Bounds(), src, src.Bounds(), draw.Over, nil)

	t := New(w, h)
	for r := 0; r < h; r++ {
		for c := 0; c < w; c++ {
			rr, gg, bb, aa := rgba.At(c, r).RGBA()
			if opts.IgnoreTransparent && aa == 0 {
				continue
			}
			lum := (299*rr + 587*gg + 114*bb) / 1000
			if lum < opts.Threshold {
				t.Set(r, c, PixelBlack)
			}
		}
	}
	return t
}
