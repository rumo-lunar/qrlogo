package main

import (
	"errors"
	"flag"
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"os"

	"github.com/rumo-lunar/qrlogo/engine"
	"github.com/rumo-lunar/qrlogo/qr/spec"
)

// Exit codes (also referenced from the README's "Exit codes" table).
const (
	exitInvalidArgs  = 1
	exitInvalidInput = 2
	exitEncodingFail = 3
	exitOutputFail   = 4
)

type exitError struct {
	code int
	err  error
}

func (e *exitError) Error() string { return e.err.Error() }
func (e *exitError) Unwrap() error { return e.err }

// run is the testable CLI entry point. It returns an *exitError so
// main() can map it to a meaningful exit code.
func run(args []string, stdout, stderr io.Writer) error {
	fs := flag.NewFlagSet("qrlogo", flag.ContinueOnError)
	fs.SetOutput(stderr)

	var (
		url            = fs.String("url", "", "URL to encode (required, byte mode)")
		ecStr          = fs.String("ec", "H", "Error correction level: L, M, Q, or H")
		versionN       = fs.Int("version", 0, "QR version 1..40 (0 = auto-fit smallest)")
		imagePath      = fs.String("image", "", "Optional logo image path (PNG, JPEG or GIF)")
		logoCoverage   = fs.Float64("logo-coverage", 0.18, "Logo box width as fraction of QR width, in (0, 1]")
		logoPadding    = fs.Float64("logo-padding", 0.10, "Background padding inside the logo box, fraction of box width")
		roundedFinders = fs.Bool("rounded-finders", true, "Render finder patterns with rounded corners")
		modules        = fs.String("modules", "square", "Module shape: square or dot")
		scale          = fs.Int("scale", 8, "Pixels per QR module")
		quiet          = fs.Int("quiet", 4, "Quiet zone in modules")
		out            = fs.String("out", "qrlogo.png", "Output PNG path (- for stdout)")
	)

	if err := fs.Parse(args); err != nil {
		return &exitError{code: exitInvalidArgs, err: err}
	}

	if *url == "" {
		return &exitError{code: exitInvalidArgs, err: errors.New("-url is required")}
	}
	ec, err := spec.ParseECLevel(*ecStr)
	if err != nil {
		return &exitError{code: exitInvalidArgs, err: err}
	}
	moduleShape, err := engine.ParseModuleShape(*modules)
	if err != nil {
		return &exitError{code: exitInvalidArgs, err: err}
	}
	if *versionN < 0 || *versionN > 40 {
		return &exitError{
			code: exitInvalidArgs,
			err:  fmt.Errorf("-version %d out of [0, 40]", *versionN),
		}
	}
	if *scale <= 0 {
		return &exitError{
			code: exitInvalidArgs,
			err:  fmt.Errorf("-scale must be positive, got %d", *scale),
		}
	}
	if *quiet < 0 {
		return &exitError{
			code: exitInvalidArgs,
			err:  fmt.Errorf("-quiet must be non-negative, got %d", *quiet),
		}
	}

	var logo image.Image
	if *imagePath != "" {
		f, err := os.Open(*imagePath)
		if err != nil {
			return &exitError{
				code: exitInvalidInput,
				err:  fmt.Errorf("opening -image: %w", err),
			}
		}
		defer f.Close()
		logo, _, err = image.Decode(f)
		if err != nil {
			return &exitError{
				code: exitInvalidInput,
				err:  fmt.Errorf("decoding -image: %w", err),
			}
		}
	}
	if logo != nil {
		if *logoCoverage <= 0 || *logoCoverage > 1 {
			return &exitError{
				code: exitInvalidArgs,
				err:  fmt.Errorf("-logo-coverage %v out of (0, 1]", *logoCoverage),
			}
		}
		if *logoCoverage > 0.25 {
			fmt.Fprintf(stderr,
				"qrlogo: warning: -logo-coverage %.2f > 0.25 may produce unscannable output even at EC H\n",
				*logoCoverage)
		}
	}

	res, err := engine.Encode(engine.Options{
		URL:     *url,
		EC:      ec,
		Version: spec.Version(*versionN),
	})
	if err != nil {
		return &exitError{code: exitEncodingFail, err: err}
	}

	w, closer, err := openOutput(*out, stdout)
	if err != nil {
		return &exitError{code: exitOutputFail, err: err}
	}
	defer closer()

	opts := engine.PNGOptions{
		Scale:         *scale,
		QuietZone:     *quiet,
		SquareFinders: !*roundedFinders,
		ModuleShape:   moduleShape,
		Logo:          logo,
		LogoCoverage:  *logoCoverage,
		LogoPadding:   *logoPadding,
	}
	if err := res.EncodePNG(w, opts); err != nil {
		return &exitError{code: exitOutputFail, err: err}
	}
	return nil
}

// openOutput resolves the -out flag to a writer and a closer the
// caller must defer. "-" routes to stdout (closer is a no-op).
func openOutput(path string, stdout io.Writer) (io.Writer, func(), error) {
	if path == "-" {
		return stdout, func() {}, nil
	}
	f, err := os.Create(path)
	if err != nil {
		return nil, nil, fmt.Errorf("creating -out: %w", err)
	}
	return f, func() { _ = f.Close() }, nil
}
