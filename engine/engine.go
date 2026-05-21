// Package engine is the integration layer of the qrlogo pipeline.
//
// It ties together the four lower-level packages:
//
//   - /qr     produces a symbolic ghost grid for V40-M mask 2,
//     plus a concrete grid of the spec-forced function-pattern bits.
//   - /render produces a visual target map whose cells say
//     "this module must be Black", "must be White", or "don't care".
//   - /bitset solves the resulting GF(2) linear system for the free
//     padding variables.
//
// Synthesize walks the symbolic grid against the target map, builds
// one bitset.Row per data-cell constraint, asks /bitset to solve,
// and substitutes the resulting bits back into the symbolic forms to
// obtain the final concrete module grid. EncodePNG renders the
// concrete grid as a scaled grayscale PNG with an optional quiet zone.
package engine

import (
	"crypto/sha256"
	"encoding/binary"
	"fmt"

	"github.com/rumo-lunar/qrlogo/bitset"
	"github.com/rumo-lunar/qrlogo/qr"
	"github.com/rumo-lunar/qrlogo/render"
)

// Options configure a single synthesis run.
type Options struct {
	// URL is the byte-mode payload encoded into the QR symbol.
	// It must be 1..qr.MaxURLBytes (2331) bytes long.
	URL string

	// Target is an optional visual constraint map sized to match the
	// QR symbol (177×177 for V40-M). nil means no constraints, in
	// which case the solver simply assigns the default free-variable
	// value (zero) and the result is a plain QR symbol carrying URL.
	Target *render.TargetMap

	// BestEffort, when true, uses SolveBestEffort instead of Solve.
	// Contradicting constraint rows are silently dropped; the result
	// approximates the target as closely as the free variables allow.
	// Use this when the logo is dense or spans the full grid.
	BestEffort bool
}

// Stats reports counters from a synthesis run.
type Stats struct {
	// FreeVars is the total number of GF(2) padding variables.
	FreeVars int

	// DataConstraints is the number of bitset.Rows added from the
	// target map's Black/White cells that landed on data modules.
	DataConstraints int

	// FunctionConflicts is the number of target Black/White cells
	// that landed on a function-pattern cell whose spec-forced bit
	// has the opposite polarity. These constraints are unsatisfiable
	// and silently ignored — they appear as "blemishes" in the
	// final image where the QR finders/timing/etc. override the
	// requested visual.
	FunctionConflicts int

	// FunctionAlignments is the number of target Black/White cells
	// that landed on a function-pattern cell whose spec-forced bit
	// happens to match — these constraints are trivially satisfied
	// by the spec and contribute no rows to the system.
	FunctionAlignments int

	// SkippedConflicts is the number of data-constraint rows that were
	// dropped by SolveBestEffort because they contradicted the system.
	// Always 0 when BestEffort is false.
	SkippedConflicts int
}

// Result is the output of one synthesis call.
type Result struct {
	// Symbol is the final 177×177 module grid (1 = dark, 0 = light).
	// It includes both data and function modules and already has
	// mask 2 baked in.
	Symbol [][]byte

	// Stats is the counters from this run.
	Stats Stats
}

// Synthesize runs the full pipeline and returns the resolved module
// grid (plus stats and the raw solver output).
//
// The returned error is non-nil only when the constraint system is
// internally inconsistent — for example, two data-cell constraints
// that demand contradictory values for the same combination of free
// variables. Conflicts against function-pattern cells are NOT errors;
// they are silently counted in Stats.FunctionConflicts so the caller
// can decide whether the visual quality is acceptable.
func Synthesize(opts Options) (*Result, error) {
	if opts.URL == "" {
		return nil, fmt.Errorf("engine: empty URL")
	}
	if len(opts.URL) > qr.MaxURLBytes {
		return nil, fmt.Errorf("engine: URL %d bytes exceeds V40-M budget %d",
			len(opts.URL), qr.MaxURLBytes)
	}

	// 1. Symbolic QR pipeline.
	codewords, d := qr.EncodeData(opts.URL)
	all := qr.Interleave(d, codewords)
	m := qr.NewMap()
	ghost := qr.PlaceCodewords(d, m, all)
	masked := qr.ApplyMask2(d, m, ghost)
	function := qr.FunctionBitsFor(m)

	stats := Stats{FreeVars: d.NumVars}

	// 2. Build the constraint system from the target map.
	sys := &bitset.System{NumVars: d.NumVars, Seed: noiseSeed(opts.URL, (d.NumVars+7)/8)}
	if opts.Target != nil {
		if opts.Target.W != m.Size || opts.Target.H != m.Size {
			return nil, fmt.Errorf("engine: target size %dx%d, want %dx%d",
				opts.Target.W, opts.Target.H, m.Size, m.Size)
		}
		opts.Target.ForEachConstraint(func(r, c int, wantBit byte) {
			if m.KindAt(r, c) != qr.KindData {
				if function[r][c] != wantBit {
					stats.FunctionConflicts++
				} else {
					stats.FunctionAlignments++
				}
				return
			}
			b := masked[r][c]
			sys.Rows = append(sys.Rows, bitset.Row{
				Vars:   append([]uint64(nil), b.Vars...),
				Target: (wantBit ^ b.Const) & 1,
			})
			stats.DataConstraints++
		})
	}

	// 3. Solve.
	var solution []byte
	if d.NumVars > 0 {
		if opts.BestEffort {
			var dropped int
			solution, dropped = sys.SolveBestEffort()
			stats.SkippedConflicts = dropped
		} else {
			var conflict int
			var ok bool
			solution, conflict, ok = sys.Solve()
			if !ok {
				return nil, fmt.Errorf(
					"engine: constraint row %d is inconsistent with the existing rows",
					conflict)
			}
		}
	}

	// 4. Resolve every cell to a concrete bit.
	grid := make([][]byte, m.Size)
	for r := 0; r < m.Size; r++ {
		grid[r] = make([]byte, m.Size)
		for c := 0; c < m.Size; c++ {
			if m.KindAt(r, c) == qr.KindData {
				grid[r][c] = d.ResolveBit(masked[r][c], solution)
			} else {
				grid[r][c] = function[r][c]
			}
		}
	}

	return &Result{
		Symbol: grid,
		Stats:  stats,
	}, nil
}

// noiseSeed expands url into n bytes of deterministic pseudorandom noise
// using iterated SHA-256 keyed by the URL. The output is stable: the same
// URL always produces the same noise, so the generated QR image is
// reproducible.
func noiseSeed(url string, n int) []byte {
	noise := make([]byte, n)
	var counter [4]byte
	offset := 0
	for offset < n {
		binary.BigEndian.PutUint32(counter[:], uint32(offset/sha256.Size))
		h := sha256.New()
		h.Write([]byte(url))
		h.Write(counter[:])
		offset += copy(noise[offset:], h.Sum(nil))
	}
	return noise
}
