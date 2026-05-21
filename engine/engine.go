// Package engine is the integration layer of the qrlogo pipeline.
//
// It ties together the qr encoder (which produces a square 0/1 module
// grid for a given URL + Spec) and the PNG renderer (which paints the
// grid, the rounded finder patterns, the quiet zone, and an optional
// centred logo overlay).
//
// Phase 1 stub: types only; Encode is intentionally not implemented
// yet — that lands in Phase 2 alongside the spec-driven qr encoder.
package engine

import (
	"fmt"

	"github.com/rumo-lunar/qrlogo/qr"
	"github.com/rumo-lunar/qrlogo/qr/spec"
)

// Options configure a single Encode call.
type Options struct {
	// URL is the byte-mode payload encoded into the QR symbol.
	URL string

	// EC is the error-correction level. Defaults to spec.ECHigh
	// when zero, because the typical use case is a centred logo
	// overlay that eats into the EC budget.
	EC spec.ECLevel

	// Version pins the QR version. Zero means "auto-fit": the
	// smallest version at the given EC level that holds URL.
	Version spec.Version
}

// Result is the output of one Encode call.
type Result struct {
	// Symbol is the final NxN module grid (1 = dark, 0 = light),
	// including every function pattern and with the chosen mask
	// already applied. Side length = Spec.Version.Size().
	Symbol [][]byte

	// Spec records the (Version, EC) actually used.
	Spec spec.Spec

	// Mask is the data-mask pattern that scored lowest by the
	// ISO/IEC 18004 §7.8.3 penalty rules. Range [0, 7].
	Mask int
}

// Encode is the main entry point. It resolves the (Version, EC)
// spec — auto-fitting the smallest version when opts.Version == 0 —
// runs the encoder, and returns the masked grid.
func Encode(opts Options) (*Result, error) {
	if opts.URL == "" {
		return nil, fmt.Errorf("engine: empty URL")
	}
	ec := opts.EC
	if ec == 0 {
		// spec.ECLow is the zero value; we instead default to ECHigh
		// because the typical use case is a logo overlay that eats
		// into the EC budget. Callers wanting L must say so.
		ec = spec.ECHigh
	}
	payload := []byte(opts.URL)

	version := opts.Version
	if version == 0 {
		v, err := spec.AutoFit(len(payload), ec)
		if err != nil {
			return nil, fmt.Errorf("engine: %w", err)
		}
		version = v
	}

	s, err := spec.New(version, ec)
	if err != nil {
		return nil, fmt.Errorf("engine: %w", err)
	}

	sym, err := qr.Build(payload, s)
	if err != nil {
		return nil, fmt.Errorf("engine: %w", err)
	}
	return &Result{Symbol: sym.Grid, Spec: s, Mask: sym.Mask}, nil
}
