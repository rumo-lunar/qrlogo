package render

import (
	"image"

	"golang.org/x/image/draw"
)

// ImageOptions configure FromImage's thresholding behaviour.
type ImageOptions struct {
	// Threshold is the luminance cutoff in the range [0, 65535].
	// Pixels with luminance strictly less than Threshold become
	// PixelBlack. Pixels at or above Threshold stay PixelDontCare
	// by default; set OpaqueBackground to map them to PixelWhite
	// instead (use this if your source image has a meaningful
	// background you want preserved as a forced light region).
	//
	// Zero means use 0x8000 (mid-grey).
	Threshold uint32

	// OpaqueBackground, when true, converts pixels above the
	// threshold into PixelWhite. When false (default), they stay
	// PixelDontCare and the QR solver is free to pick.
	OpaqueBackground bool

	// IgnoreTransparent, when true, leaves source pixels with zero
	// alpha as PixelDontCare regardless of any other rule. This is
	// what you want for logos with a transparent background.
	IgnoreTransparent bool

	// Scaler controls the downsampling kernel. If nil,
	// draw.CatmullRom is used (sharp but smooth).
	Scaler draw.Scaler
}

// FromImage downsamples src to w×h pixels and thresholds each pixel
// into a PixelState. The result has only PixelBlack and PixelDontCare
// (or PixelWhite, when OpaqueBackground is set).
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
	scaler := opts.Scaler
	if scaler == nil {
		scaler = draw.CatmullRom
	}

	rgba := image.NewRGBA(image.Rect(0, 0, w, h))
	scaler.Scale(rgba, rgba.Bounds(), src, src.Bounds(), draw.Over, nil)

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
			} else if opts.OpaqueBackground {
				t.Set(r, c, PixelWhite)
			}
		}
	}
	return t
}
