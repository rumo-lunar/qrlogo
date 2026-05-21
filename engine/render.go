package engine

import (
	"image"
	"image/color"
	"math"

	"github.com/rumo-lunar/qrlogo/qr/spec"
	"golang.org/x/image/draw"
)

// renderSymbol rasterises a QR module grid into an *image.RGBA at the
// requested scale and quiet-zone padding.
//
// Three orthogonal style switches:
//
//   - roundedFinders == true   draws the three 7×7 finder patterns as
//     rounded shapes (outer ring + inner dot) instead of plain
//     squares. Always recommended; scanners locate finders by their
//     1:1:3:1:1 ratio which the rounded form preserves.
//
//   - dotModules == true       renders every non-finder module as a
//     filled circle (data, timing, alignment, dark, version,
//     format). Real-world dot QR codes rely on EC level H to
//     compensate for the slightly reduced ink coverage of circles
//     versus squares.
//
//   - reserved                 a pixel-coordinate rectangle that
//     modules are skipped inside of. Used to clear the area behind
//     a centred logo so circles aren't drawn half-overlapped by the
//     overlay. Pass image.Rectangle{} (zero value) for no clearing.
func renderSymbol(
	grid [][]byte,
	v spec.Version,
	scale, quiet int,
	fg, bg color.RGBA,
	roundedFinders, dotModules bool,
	reserved image.Rectangle,
) *image.RGBA {
	n := len(grid)
	size := (n + 2*quiet) * scale
	img := image.NewRGBA(image.Rect(0, 0, size, size))

	// Background fill.
	for y := 0; y < size; y++ {
		for x := 0; x < size; x++ {
			img.SetRGBA(x, y, bg)
		}
	}

	// Mark cells that belong to a finder pattern so the main loop
	// skips them — they are drawn separately as rounded shapes.
	inFinder := make([][]bool, n)
	for r := 0; r < n; r++ {
		inFinder[r] = make([]bool, n)
	}
	finderOrigins := v.FinderOrigins()
	if roundedFinders {
		for _, o := range finderOrigins {
			for dr := 0; dr < 7; dr++ {
				for dc := 0; dc < 7; dc++ {
					inFinder[o[0]+dr][o[1]+dc] = true
				}
			}
		}
	}

	// Per-module rendering. In dot mode every non-finder module is a
	// circle, including alignment patterns — at EC H this is the
	// trade we accept for visual consistency.
	rDot := float64(scale) / 2
	hasReserved := !reserved.Empty()
	for r := 0; r < n; r++ {
		for c := 0; c < n; c++ {
			if grid[r][c] == 0 || inFinder[r][c] {
				continue
			}
			x0 := (c + quiet) * scale
			y0 := (r + quiet) * scale
			// Skip any module whose centre falls inside the
			// reserved rect (the logo footprint). The EC budget
			// absorbs the cleared cells.
			if hasReserved {
				cxp := x0 + scale/2
				cyp := y0 + scale/2
				if (image.Point{X: cxp, Y: cyp}).In(reserved) {
					continue
				}
			}
			if dotModules {
				cxp := float64(x0) + rDot
				cyp := float64(y0) + rDot
				fillCircle(img, cxp, cyp, rDot, fg)
				continue
			}
			fillRect(img, x0, y0, x0+scale, y0+scale, fg)
		}
	}

	// Rounded finder patterns.
	if roundedFinders {
		for _, o := range finderOrigins {
			drawRoundedFinder(img, o[0], o[1], quiet, scale, fg, bg)
		}
	}
	return img
}

// drawRoundedFinder paints a rounded version of the 7×7 finder
// pattern whose top-left module is (mr, mc). It draws:
//
//   - an outer 7×7 dark rounded square (corner radius ≈ 2 modules);
//   - a 5×5 background-coloured rounded square inset by 1 module
//     (corner radius ≈ 1.5 modules) to carve out the light ring;
//   - a 3×3 dark rounded square inset by 2 modules
//     (corner radius ≈ 1 module) for the centre dot.
func drawRoundedFinder(img *image.RGBA, mr, mc, quiet, scale int, fg, bg color.RGBA) {
	s := float64(scale)
	x0 := float64((mc + quiet) * scale)
	y0 := float64((mr + quiet) * scale)

	// Outer 7×7 dark.
	fillRoundedRect(img, x0, y0, x0+7*s, y0+7*s, 2*s, fg)
	// Inner 5×5 light (carves the ring).
	fillRoundedRect(img, x0+s, y0+s, x0+6*s, y0+6*s, 1.5*s, bg)
	// Inner 3×3 dark dot.
	fillRoundedRect(img, x0+2*s, y0+2*s, x0+5*s, y0+5*s, s, fg)
}

// fillRect fills an axis-aligned rectangle in img with c. No AA.
func fillRect(img *image.RGBA, x0, y0, x1, y1 int, c color.RGBA) {
	for y := y0; y < y1; y++ {
		for x := x0; x < x1; x++ {
			img.SetRGBA(x, y, c)
		}
	}
}

// fillRoundedRect fills a rounded rectangle [(x0,y0)-(x1,y1)] with
// corner radius r and colour c, using a signed-distance-field
// coverage calculation for 1-pixel anti-aliasing along the curve.
func fillRoundedRect(img *image.RGBA, x0, y0, x1, y1, r float64, c color.RGBA) {
	// Bounding box in pixel coords, clipped to the image.
	xMin := int(math.Floor(x0)) - 1
	yMin := int(math.Floor(y0)) - 1
	xMax := int(math.Ceil(x1)) + 1
	yMax := int(math.Ceil(y1)) + 1
	b := img.Bounds()
	if xMin < b.Min.X {
		xMin = b.Min.X
	}
	if yMin < b.Min.Y {
		yMin = b.Min.Y
	}
	if xMax > b.Max.X {
		xMax = b.Max.X
	}
	if yMax > b.Max.Y {
		yMax = b.Max.Y
	}

	for y := yMin; y < yMax; y++ {
		py := float64(y) + 0.5
		for x := xMin; x < xMax; x++ {
			px := float64(x) + 0.5
			a := roundedRectCoverage(px, py, x0, y0, x1, y1, r)
			if a <= 0 {
				continue
			}
			if a >= 1 {
				img.SetRGBA(x, y, c)
				continue
			}
			// Source-over blend: out = src*a + dst*(1-a).
			dst := img.RGBAAt(x, y)
			img.SetRGBA(x, y, color.RGBA{
				R: blend(c.R, dst.R, a),
				G: blend(c.G, dst.G, a),
				B: blend(c.B, dst.B, a),
				A: 255,
			})
		}
	}
}

// roundedRectCoverage returns the fraction of the pixel centred at
// (px,py) that lies inside the rounded rectangle.
//
// Uses the textbook 2D signed distance to a rounded rectangle:
//
//	1. Clamp (px,py) into the rectangle shrunk by r on every side.
//	2. d = distance(pixel, clamp) − r.
//	3. d ≤ 0 → inside; d > 0 → outside.
//	4. AA on a 1-pixel-wide band around d = 0.
func roundedRectCoverage(px, py, x0, y0, x1, y1, r float64) float64 {
	cx := clamp(px, x0+r, x1-r)
	cy := clamp(py, y0+r, y1-r)
	dx := px - cx
	dy := py - cy
	d := math.Sqrt(dx*dx+dy*dy) - r
	if d <= -0.5 {
		return 1
	}
	if d >= 0.5 {
		return 0
	}
	return 0.5 - d
}

func clamp(v, lo, hi float64) float64 {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

// fillCircle blends c into img inside a disc of radius r centred at
// (cx, cy), using the same 1-pixel SDF anti-aliasing as
// fillRoundedRect.
func fillCircle(img *image.RGBA, cx, cy, r float64, c color.RGBA) {
	xMin := int(math.Floor(cx-r)) - 1
	yMin := int(math.Floor(cy-r)) - 1
	xMax := int(math.Ceil(cx+r)) + 1
	yMax := int(math.Ceil(cy+r)) + 1
	b := img.Bounds()
	if xMin < b.Min.X {
		xMin = b.Min.X
	}
	if yMin < b.Min.Y {
		yMin = b.Min.Y
	}
	if xMax > b.Max.X {
		xMax = b.Max.X
	}
	if yMax > b.Max.Y {
		yMax = b.Max.Y
	}

	for y := yMin; y < yMax; y++ {
		py := float64(y) + 0.5
		dy := py - cy
		for x := xMin; x < xMax; x++ {
			px := float64(x) + 0.5
			dx := px - cx
			d := math.Sqrt(dx*dx+dy*dy) - r
			if d >= 0.5 {
				continue
			}
			if d <= -0.5 {
				img.SetRGBA(x, y, c)
				continue
			}
			a := 0.5 - d
			dst := img.RGBAAt(x, y)
			img.SetRGBA(x, y, color.RGBA{
				R: blend(c.R, dst.R, a),
				G: blend(c.G, dst.G, a),
				B: blend(c.B, dst.B, a),
				A: 255,
			})
		}
	}
}

// blend returns src*a + dst*(1-a) as a uint8.
func blend(src, dst uint8, a float64) uint8 {
	v := float64(src)*a + float64(dst)*(1-a)
	if v < 0 {
		v = 0
	}
	if v > 255 {
		v = 255
	}
	return uint8(v + 0.5)
}

// drawLogo composites src into img centred at (cx, cy), preserving
// the source aspect ratio. boxSize bounds the LONGER side; the
// shorter side is scaled proportionally. A bg-coloured rounded
// rectangle is painted behind the logo as a padding card so the
// artwork is legible against the surrounding QR modules.
//
// No QR modules are cleared — src is painted on top. The EC budget
// has to absorb the modules that disappear under the logo. Callers
// should pick coverage and EC level accordingly.
func drawLogo(img *image.RGBA, src image.Image, cx, cy, boxSize int, padding float64, bg color.RGBA) {
	if boxSize <= 0 {
		return
	}
	sw := src.Bounds().Dx()
	sh := src.Bounds().Dy()
	if sw <= 0 || sh <= 0 {
		return
	}

	// Scale the source so its longer side equals boxSize, preserving
	// aspect ratio. Integer division on the shorter side rounds down,
	// which keeps the rendered logo strictly inside the bounding box.
	var lw, lh int
	if sw >= sh {
		lw = boxSize
		lh = boxSize * sh / sw
	} else {
		lh = boxSize
		lw = boxSize * sw / sh
	}
	if lw < 1 {
		lw = 1
	}
	if lh < 1 {
		lh = 1
	}

	// Optional padding card: a square with side = boxSize + 2·pad,
	// independent of logo aspect ratio. Skipped entirely when
	// padding <= 0 so the logo is composited directly onto the
	// QR modules with no background fill.
	pad := int(float64(boxSize) * padding)
	if pad > 0 {
		cardHalf := boxSize/2 + pad
		fillRoundedRect(img,
			float64(cx-cardHalf), float64(cy-cardHalf),
			float64(cx+cardHalf), float64(cy+cardHalf),
			float64(pad), bg,
		)
	}

	logoX0 := cx - lw/2
	logoY0 := cy - lh/2
	logoX1 := logoX0 + lw
	logoY1 := logoY0 + lh

	// Scale src into an lw × lh RGBA buffer at full Catmull-Rom
	// quality, then alpha-blend it onto img.
	buf := image.NewRGBA(image.Rect(0, 0, lw, lh))
	draw.CatmullRom.Scale(buf, buf.Bounds(), src, src.Bounds(), draw.Over, nil)
	draw.Draw(img, image.Rect(logoX0, logoY0, logoX1, logoY1), buf, image.Point{}, draw.Over)
}
