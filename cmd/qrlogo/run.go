package main

import (
	"flag"
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"os"

	"github.com/rumo-lunar/qrlogo/engine"
	"github.com/rumo-lunar/qrlogo/qr"
	"github.com/rumo-lunar/qrlogo/render"
)

// exitError carries an exit code alongside its message.
type exitError struct {
	code int
	msg  string
}

func (e *exitError) Error() string { return e.msg }

// exitf builds an *exitError with a formatted message.
func exitf(code int, format string, args ...any) *exitError {
	return &exitError{code: code, msg: fmt.Sprintf(format, args...)}
}

func run(args []string, stdout, stderr io.Writer) error {
	fs := flag.NewFlagSet("qrlogo", flag.ContinueOnError)
	fs.SetOutput(stderr)

	urlFlag := fs.String("url", "", "byte-mode payload (required, ≤2331 bytes)")
	imageFlag := fs.String("image", "", "path to PNG/JPEG/GIF logo image")
	textFlag := fs.String("text", "", "text to embed as logo")
	outFlag := fs.String("out", "qrlogo.png", `output PNG path ("-" for stdout)`)
	scaleFlag := fs.Int("scale", 8, "pixels per QR module")
	quietFlag := fs.Int("quiet", 4, "quiet-zone modules")
	threshFlag := fs.Uint("threshold", 0x8000, "luminance cutoff for image thresholding [0,65535]")
	noHaloFlag := fs.Bool("no-halo", false, "skip 8-neighbour halo around dark logo cells")
	logoScaleFlag := fs.Float64("logo-scale", 1.0, "fraction of the QR grid the logo fills (0.0–1.0); logo is centred")
	statsFlag := fs.Bool("stats", false, "print synthesis stats to stderr")
	bestEffortFlag := fs.Bool("best-effort", false, "skip contradicting constraints instead of failing (recommended for dense logos)")

	if err := fs.Parse(args); err != nil {
		return exitf(1, "%s", err.Error())
	}

	if *urlFlag == "" {
		_, _ = fmt.Fprintln(stderr, "qrlogo: -url is required")
		fs.Usage()
		return exitf(1, "-url is required")
	}
	if len(*urlFlag) > qr.MaxURLBytes {
		return exitf(2, "qrlogo: URL is %d bytes, maximum is %d", len(*urlFlag), qr.MaxURLBytes)
	}
	if *imageFlag != "" && *textFlag != "" {
		return exitf(1, "qrlogo: -image and -text are mutually exclusive")
	}
	if *logoScaleFlag <= 0 || *logoScaleFlag > 1 {
		return exitf(1, "qrlogo: -logo-scale must be in (0, 1]")
	}

	target, err := buildTarget(*imageFlag, *textFlag, uint32(*threshFlag), *noHaloFlag, *logoScaleFlag, qr.Size)
	if err != nil {
		return exitf(2, "qrlogo: %v", err)
	}

	result, err := engine.Synthesize(engine.Options{
		URL:        *urlFlag,
		Target:     target,
		BestEffort: *bestEffortFlag,
	})
	if err != nil {
		return exitf(3, "qrlogo: synthesis failed: %v", err)
	}

	if *statsFlag {
		s := result.Stats
		_, _ = fmt.Fprintf(stderr, "free vars        : %d\n", s.FreeVars)
		_, _ = fmt.Fprintf(stderr, "data constraints : %d\n", s.DataConstraints)
		_, _ = fmt.Fprintf(stderr, "function aligns  : %d\n", s.FunctionAlignments)
		_, _ = fmt.Fprintf(stderr, "function conflicts: %d\n", s.FunctionConflicts)
		_, _ = fmt.Fprintf(stderr, "skipped conflicts: %d\n", s.SkippedConflicts)
	}

	pngOpts := engine.PNGOptions{Scale: *scaleFlag, QuietZone: *quietFlag}
	if *outFlag == "-" {
		if err := result.EncodePNG(stdout, pngOpts); err != nil {
			return exitf(4, "qrlogo: write failed: %v", err)
		}
		return nil
	}

	f, err := os.Create(*outFlag)
	if err != nil {
		return exitf(4, "qrlogo: cannot create output file: %v", err)
	}
	defer func() { _ = f.Close() }()

	if err := result.EncodePNG(f, pngOpts); err != nil {
		return exitf(4, "qrlogo: write failed: %v", err)
	}
	if err := f.Close(); err != nil {
		return exitf(4, "qrlogo: close failed: %v", err)
	}
	return nil
}

func buildTarget(imagePath, text string, threshold uint32, noHalo bool, logoScale float64, gridSize int) (*render.TargetMap, error) {
	// sub is the side length of the logo region within the grid.
	sub := int(float64(gridSize) * logoScale)
	if sub < 1 {
		sub = 1
	}
	// offset centres the sub-grid.
	offset := (gridSize - sub) / 2

	var inner *render.TargetMap

	switch {
	case imagePath != "":
		f, err := os.Open(imagePath)
		if err != nil {
			return nil, fmt.Errorf("cannot open image %q: %w", imagePath, err)
		}
		defer func() { _ = f.Close() }()

		src, _, err := image.Decode(f)
		if err != nil {
			return nil, fmt.Errorf("cannot decode image %q: %w", imagePath, err)
		}
		src = cropTransparent(src)
		inner = render.FromImage(src, sub, sub, render.ImageOptions{
			Threshold:         threshold,
			IgnoreTransparent: true,
		})

	case text != "":
		inner = render.RenderText(text, sub, sub)
	}

	if inner == nil {
		return nil, nil
	}

	if !noHalo {
		render.ApplyHalo(inner)
	}

	// If the logo fills the full grid, return it directly.
	if sub == gridSize {
		return inner, nil
	}

	// Stamp the sub-grid into a full gridSize×gridSize target (remainder stays DontCare).
	full := render.New(gridSize, gridSize)
	for r := 0; r < sub; r++ {
		for c := 0; c < sub; c++ {
			if p := inner.At(r, c); p != render.PixelDontCare {
				full.Set(offset+r, offset+c, p)
			}
		}
	}
	return full, nil
}

// cropTransparent returns a sub-image of src trimmed to the bounding box of
// its opaque pixels. Images without an alpha channel are returned unchanged.
// If all pixels are transparent, src is returned unchanged.
func cropTransparent(src image.Image) image.Image {
	b := src.Bounds()
	minX, minY := b.Max.X, b.Max.Y
	maxX, maxY := b.Min.X, b.Min.Y

	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			_, _, _, a := src.At(x, y).RGBA()
			if a == 0 {
				continue
			}
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

	if minX > maxX || minY > maxY {
		return src // all transparent — nothing to crop
	}

	type subImager interface {
		SubImage(image.Rectangle) image.Image
	}
	if si, ok := src.(subImager); ok {
		return si.SubImage(image.Rect(minX, minY, maxX+1, maxY+1))
	}
	return src
}
