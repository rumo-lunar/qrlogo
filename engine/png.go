package engine

import (
	"fmt"
	"image"
	"image/color"
	"image/png"
	"io"
)

// ModuleShape selects how individual data modules are rasterised.
//
// Finder patterns (always) and alignment patterns (in Dot mode)
// remain solid regardless of ModuleShape — their detectability
// depends on contiguous shapes.
type ModuleShape int

const (
	// ModuleShapeSquare renders every module as a filled square.
	// This is the zero value and the default.
	ModuleShapeSquare ModuleShape = iota

	// ModuleShapeDot renders data and timing modules as filled
	// circles. Alignment patterns are kept solid.
	ModuleShapeDot
)

// String returns the lower-case label used by the CLI -modules flag.
func (s ModuleShape) String() string {
	switch s {
	case ModuleShapeSquare:
		return "square"
	case ModuleShapeDot:
		return "dot"
	}
	return "?"
}

// ParseModuleShape parses a CLI label ("square" or "dot") into a
// ModuleShape, case-insensitive.
func ParseModuleShape(s string) (ModuleShape, error) {
	switch s {
	case "square", "Square", "SQUARE":
		return ModuleShapeSquare, nil
	case "dot", "Dot", "DOT":
		return ModuleShapeDot, nil
	}
	return 0, fmt.Errorf("engine: unknown module shape %q (want square or dot)", s)
}

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

	// ModuleShape selects the shape of data and timing modules.
	// The zero value (ModuleShapeSquare) renders every module as a
	// square, matching the historical look.
	ModuleShape ModuleShape

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

	// Compute the logo footprint up front so the renderer can skip
	// every module whose centre falls inside it. The footprint is
	// the square padding card (boxSize + 2·pad on a side), centred
	// at the QR centre. With LogoPadding == 0 it shrinks to just the
	// logo's bounding square.
	var (
		reserved image.Rectangle
		boxSize  int
		logoCX   int
		logoCY   int
	)
	if o.Logo != nil {
		symPx := n * o.Scale
		boxSize = int(float64(symPx) * o.LogoCoverage)
		if boxSize < 1 {
			boxSize = 1
		}
		logoCX = o.QuietZone*o.Scale + symPx/2
		logoCY = logoCX
		pad := int(float64(boxSize) * o.LogoPadding)
		cardHalf := boxSize/2 + pad
		reserved = image.Rect(
			logoCX-cardHalf, logoCY-cardHalf,
			logoCX+cardHalf, logoCY+cardHalf,
		)
	}

	img := renderSymbol(
		r.Symbol,
		r.Spec.Version,
		o.Scale,
		o.QuietZone,
		fg, bg,
		!o.SquareFinders,
		o.ModuleShape == ModuleShapeDot,
		reserved,
	)

	if o.Logo != nil {
		drawLogo(img, o.Logo, logoCX, logoCY, boxSize, o.LogoPadding, bg)
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
