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

func run(args []string, stdout, stderr io.Writer) error {
	fs := flag.NewFlagSet("qrlogo", flag.ContinueOnError)
	fs.SetOutput(stderr)

	urlFlag := fs.String("url", "", "byte-mode payload (required, ≤100 bytes)")
	imageFlag := fs.String("image", "", "path to PNG/JPEG/GIF logo image")
	textFlag := fs.String("text", "", "text to embed as logo")
	outFlag := fs.String("out", "qrlogo.png", `output PNG path ("-" for stdout)`)
	scaleFlag := fs.Int("scale", 8, "pixels per QR module")
	quietFlag := fs.Int("quiet", 4, "quiet-zone modules")
	threshFlag := fs.Uint("threshold", 0x8000, "luminance cutoff for image thresholding [0,65535]")
	noHaloFlag := fs.Bool("no-halo", false, "skip 8-neighbour halo around dark logo cells")
	statsFlag := fs.Bool("stats", false, "print synthesis stats to stderr")

	if err := fs.Parse(args); err != nil {
		return &exitError{code: 1, msg: err.Error()}
	}

	if *urlFlag == "" {
		fmt.Fprintln(stderr, "qrlogo: -url is required")
		fs.Usage()
		return &exitError{code: 1, msg: "-url is required"}
	}
	if len(*urlFlag) > qr.MaxURLBytesV11M {
		return &exitError{
			code: 2,
			msg:  fmt.Sprintf("qrlogo: URL is %d bytes, maximum is %d", len(*urlFlag), qr.MaxURLBytesV11M),
		}
	}
	if *imageFlag != "" && *textFlag != "" {
		return &exitError{code: 1, msg: "qrlogo: -image and -text are mutually exclusive"}
	}

	target, err := buildTarget(*imageFlag, *textFlag, uint32(*threshFlag), *noHaloFlag)
	if err != nil {
		return &exitError{code: 2, msg: fmt.Sprintf("qrlogo: %v", err)}
	}

	result, err := engine.Synthesize(engine.Options{
		URL:    *urlFlag,
		Target: target,
	})
	if err != nil {
		return &exitError{code: 3, msg: fmt.Sprintf("qrlogo: synthesis failed: %v", err)}
	}

	if *statsFlag {
		s := result.Stats
		fmt.Fprintf(stderr, "free vars        : %d\n", s.FreeVars)
		fmt.Fprintf(stderr, "data constraints : %d\n", s.DataConstraints)
		fmt.Fprintf(stderr, "function aligns  : %d\n", s.FunctionAlignments)
		fmt.Fprintf(stderr, "function conflicts: %d\n", s.FunctionConflicts)
	}

	pngOpts := engine.PNGOptions{Scale: *scaleFlag, QuietZone: *quietFlag}
	if *outFlag == "-" {
		if err := result.EncodePNG(stdout, pngOpts); err != nil {
			return &exitError{code: 4, msg: fmt.Sprintf("qrlogo: write failed: %v", err)}
		}
		return nil
	}

	f, err := os.Create(*outFlag)
	if err != nil {
		return &exitError{code: 4, msg: fmt.Sprintf("qrlogo: cannot create output file: %v", err)}
	}
	defer f.Close()

	if err := result.EncodePNG(f, pngOpts); err != nil {
		return &exitError{code: 4, msg: fmt.Sprintf("qrlogo: write failed: %v", err)}
	}
	return nil
}

func buildTarget(imagePath, text string, threshold uint32, noHalo bool) (*render.TargetMap, error) {
	var target *render.TargetMap

	switch {
	case imagePath != "":
		f, err := os.Open(imagePath)
		if err != nil {
			return nil, fmt.Errorf("cannot open image %q: %w", imagePath, err)
		}
		defer f.Close()

		src, _, err := image.Decode(f)
		if err != nil {
			return nil, fmt.Errorf("cannot decode image %q: %w", imagePath, err)
		}
		target = render.FromImage(src, 61, 61, render.ImageOptions{
			Threshold:         threshold,
			IgnoreTransparent: true,
		})

	case text != "":
		target = render.RenderText(text, 61, 61, render.TextOptions{})
	}

	if target != nil && !noHalo {
		render.ApplyHalo(target)
	}
	return target, nil
}
