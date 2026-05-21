package engine

import (
	"fmt"
	"image"
	"image/color"
	"image/png"
	"io"
)

// PNGOptions configure how a Result is rendered as a PNG image.
//
// Zero values produce a sensible default: black-on-white, scale 8,
// quiet zone 4, rounded finder patterns enabled, no logo overlay.
type PNGOptions struct {
	// Scale is the side length in pixels of a single QR module.
	// Zero means 8.
	Scale int

	// QuietZone is the width in modules of the light border around
	// the symbol. Zero means 4 (the ISO/IEC 18004 minimum).
	QuietZone int

	// Foreground is the colour of dark modules. nil means black.
	Foreground color.Color

	// Background is the colour of light modules and the quiet zone.
	// nil means white.
	Background color.Color

	// SquareFinders disables the rounded finder treatment that is
	// applied by default. Zero value (false) keeps the rounded look.
	SquareFinders bool

	// Logo is an optional image painted on top of the rendered QR,
	// centred in the symbol. No QR modules are cleared — the error-
	// correction budget has to absorb the obscured modules.
	Logo image.Image

	// LogoCoverage bounds the LONGER side of the logo as a fraction
	// of the QR symbol (excluding quiet zone), in (0, 1]. The shorter
	// side scales proportionally so the source aspect ratio is
	// preserved. Anything past about 0.25 risks unscannable output
	// even at EC H. Zero means 0.18 when Logo is set, 0 otherwise.
	LogoCoverage float64

	// LogoPadding is the padding rendered as a solid Background
	// rectangle behind the logo, as a fraction of the logo box.
	// Zero (the default) means no padding card is drawn — the logo
	// is composited directly on top of the QR modules.
	LogoPadding float64
}

func (o PNGOptions) resolved() PNGOptions {
	if o.Scale == 0 {
		o.Scale = 8
	}
	if o.QuietZone == 0 {
		o.QuietZone = 4
	}
	if o.Foreground == nil {
		o.Foreground = color.Black
	}
	if o.Background == nil {
		o.Background = color.White
	}
	if o.Logo != nil && o.LogoCoverage == 0 {
		o.LogoCoverage = 0.18
	}
	// LogoPadding is intentionally NOT defaulted: zero means
	// "no padding card", which is a useful behaviour rather than
	// an unset sentinel.
	return o
}

// EncodePNG renders r as a PNG and writes it to w.
//
// Returns an error if r.Symbol is empty or non-square, if opts is
// malformed (e.g. negative scale), or if PNG encoding itself fails.
//
// EncodePNG warns on stderr when LogoCoverage > 0.25 because real
// scanners start failing past that threshold even at EC H.
func (r *Result) EncodePNG(w io.Writer, opts PNGOptions) error {
	if len(r.Symbol) == 0 {
		return fmt.Errorf("engine: empty symbol")
	}
	n := len(r.Symbol)
	for _, row := range r.Symbol {
		if len(row) != n {
			return fmt.Errorf("engine: non-square symbol")
		}
	}

	o := opts.resolved()
	if o.Scale <= 0 {
		return fmt.Errorf("engine: scale must be positive, got %d", o.Scale)
	}
	if o.QuietZone < 0 {
		return fmt.Errorf("engine: quiet zone must be non-negative, got %d", o.QuietZone)
	}
	if o.Logo != nil && (o.LogoCoverage <= 0 || o.LogoCoverage > 1) {
		return fmt.Errorf("engine: logo coverage %v out of (0, 1]", o.LogoCoverage)
	}

	fg := toRGBA(o.Foreground)
	bg := toRGBA(o.Background)

	img := renderSymbol(
		r.Symbol,
		r.Spec.Version.FinderOrigins(),
		o.Scale,
		o.QuietZone,
		fg, bg,
		!o.SquareFinders,
	)

	if o.Logo != nil {
		symPx := n * o.Scale
		boxSize := int(float64(symPx) * o.LogoCoverage)
		if boxSize < 1 {
			boxSize = 1
		}
		cx := o.QuietZone*o.Scale + symPx/2
		cy := o.QuietZone*o.Scale + symPx/2
		drawLogo(img, o.Logo, cx, cy, boxSize, o.LogoPadding, bg)
	}

	return png.Encode(w, img)
}

// toRGBA converts a color.Color to color.RGBA via the standard
// 16-bit channel intermediate.
func toRGBA(c color.Color) color.RGBA {
	r, g, b, a := c.RGBA()
	return color.RGBA{
		R: uint8(r >> 8),
		G: uint8(g >> 8),
		B: uint8(b >> 8),
		A: uint8(a >> 8),
	}
}
