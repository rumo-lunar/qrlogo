package engine_test

import (
	"bytes"
	"image/png"
	"testing"

	"github.com/rumo-lunar/qrlogo/engine"
	"github.com/rumo-lunar/qrlogo/qr"
	"github.com/rumo-lunar/qrlogo/render"
)

func TestSynthesize_NoTarget_ProducesV40Grid(t *testing.T) {
	res, err := engine.Synthesize(engine.Options{URL: "https://example.com"})
	if err != nil {
		t.Fatalf("Synthesize: %v", err)
	}
	if len(res.Symbol) != 177 {
		t.Fatalf("rows = %d, want 177", len(res.Symbol))
	}
	for r, row := range res.Symbol {
		if len(row) != 177 {
			t.Errorf("row %d width = %d, want 177", r, len(row))
		}
		for c, v := range row {
			if v != 0 && v != 1 {
				t.Errorf("cell (%d,%d) = %d, want 0 or 1", r, c, v)
			}
		}
	}
}

func TestSynthesize_NoTarget_FunctionPatternsMatchSpec(t *testing.T) {
	res, err := engine.Synthesize(engine.Options{URL: "x"})
	if err != nil {
		t.Fatalf("Synthesize: %v", err)
	}
	m := qr.NewMap()
	fn := qr.FunctionBits()
	for r := 0; r < m.Size; r++ {
		for c := 0; c < m.Size; c++ {
			if m.KindAt(r, c) == qr.KindData {
				continue
			}
			if res.Symbol[r][c] != fn[r][c] {
				t.Errorf("function cell (%d,%d) = %d, spec = %d",
					r, c, res.Symbol[r][c], fn[r][c])
			}
		}
	}
}

func TestSynthesize_NoTarget_StatsZero(t *testing.T) {
	res, err := engine.Synthesize(engine.Options{URL: "x"})
	if err != nil {
		t.Fatalf("Synthesize: %v", err)
	}
	if res.Stats.DataConstraints != 0 {
		t.Errorf("DataConstraints = %d, want 0", res.Stats.DataConstraints)
	}
	if res.Stats.FunctionConflicts != 0 {
		t.Errorf("FunctionConflicts = %d, want 0", res.Stats.FunctionConflicts)
	}
	// URL "x" → 1 byte URL → (2334 - 1 - 3)*8 free vars.
	if got, want := res.Stats.FreeVars, (qr.DataCodewords-1-3)*8; got != want {
		t.Errorf("FreeVars = %d, want %d", got, want)
	}
}

func TestSynthesize_TargetConstraintsAreSatisfied(t *testing.T) {
	// Pick a handful of data-region cells far from function patterns
	// and force them to specific values; verify they come out that way.
	target := render.New(177, 177)
	m := qr.NewMap()
	type want struct {
		r, c int
		bit  byte
	}
	picks := []want{
		{15, 15, 1},
		{15, 16, 0},
		{20, 25, 1},
		{30, 40, 0},
		{40, 20, 1},
	}
	for _, p := range picks {
		if m.KindAt(p.r, p.c) != qr.KindData {
			t.Fatalf("test setup: (%d,%d) is not a data cell", p.r, p.c)
		}
		if p.bit == 1 {
			target.Set(p.r, p.c, render.PixelBlack)
		} else {
			target.Set(p.r, p.c, render.PixelWhite)
		}
	}

	res, err := engine.Synthesize(engine.Options{URL: "https://lunar.app", Target: target})
	if err != nil {
		t.Fatalf("Synthesize: %v", err)
	}
	for _, p := range picks {
		if got := res.Symbol[p.r][p.c]; got != p.bit {
			t.Errorf("cell (%d,%d) = %d, want %d", p.r, p.c, got, p.bit)
		}
	}
	if res.Stats.DataConstraints != len(picks) {
		t.Errorf("DataConstraints = %d, want %d", res.Stats.DataConstraints, len(picks))
	}
}

func TestSynthesize_TargetOnFunctionCellAlignsOrConflicts(t *testing.T) {
	// (0,0) is the top-left finder corner — spec says dark (=1).
	// Asking for Black aligns; asking for White conflicts.
	fn := qr.FunctionBits()
	if fn[0][0] != 1 {
		t.Fatalf("test assumption broke: fn[0][0] = %d", fn[0][0])
	}

	tAlign := render.New(177, 177)
	tAlign.Set(0, 0, render.PixelBlack)
	res, err := engine.Synthesize(engine.Options{URL: "x", Target: tAlign})
	if err != nil {
		t.Fatalf("aligned: %v", err)
	}
	if res.Stats.FunctionAlignments != 1 || res.Stats.FunctionConflicts != 0 {
		t.Errorf("aligned stats = %+v, want 1 alignment / 0 conflict", res.Stats)
	}
	if res.Symbol[0][0] != 1 {
		t.Errorf("aligned symbol[0][0] = %d, want 1", res.Symbol[0][0])
	}

	tConflict := render.New(177, 177)
	tConflict.Set(0, 0, render.PixelWhite)
	res, err = engine.Synthesize(engine.Options{URL: "x", Target: tConflict})
	if err != nil {
		t.Fatalf("conflict: %v", err)
	}
	if res.Stats.FunctionConflicts != 1 || res.Stats.FunctionAlignments != 0 {
		t.Errorf("conflict stats = %+v, want 0 alignment / 1 conflict", res.Stats)
	}
	if res.Symbol[0][0] != 1 {
		t.Errorf("conflict symbol[0][0] = %d, want 1 (spec wins)", res.Symbol[0][0])
	}
}

func TestSynthesize_RejectsEmptyURL(t *testing.T) {
	_, err := engine.Synthesize(engine.Options{URL: ""})
	if err == nil {
		t.Error("empty URL did not error")
	}
}

func TestSynthesize_RejectsOversizedURL(t *testing.T) {
	long := bytes.Repeat([]byte{'a'}, qr.MaxURLBytes+1)
	_, err := engine.Synthesize(engine.Options{URL: string(long)})
	if err == nil {
		t.Error("oversized URL did not error")
	}
}

func TestSynthesize_RejectsWrongTargetSize(t *testing.T) {
	target := render.New(20, 20)
	_, err := engine.Synthesize(engine.Options{URL: "x", Target: target})
	if err == nil {
		t.Error("wrong target size did not error")
	}
}

func TestEncodePNG_Defaults(t *testing.T) {
	res, err := engine.Synthesize(engine.Options{URL: "https://example.com"})
	if err != nil {
		t.Fatalf("Synthesize: %v", err)
	}
	var buf bytes.Buffer
	if err := res.EncodePNG(&buf, engine.PNGOptions{}); err != nil {
		t.Fatalf("EncodePNG: %v", err)
	}
	img, err := png.Decode(&buf)
	if err != nil {
		t.Fatalf("png.Decode: %v", err)
	}
	// Default scale=8, quietZone=4 → (177 + 8) * 8 px per side.
	want := (177 + 8) * 8
	b := img.Bounds()
	if b.Dx() != want || b.Dy() != want {
		t.Errorf("size = %dx%d, want %dx%d", b.Dx(), b.Dy(), want, want)
	}
}

func TestEncodePNG_CustomScaleAndQuiet(t *testing.T) {
	res, err := engine.Synthesize(engine.Options{URL: "https://example.com"})
	if err != nil {
		t.Fatalf("Synthesize: %v", err)
	}
	var buf bytes.Buffer
	if err := res.EncodePNG(&buf, engine.PNGOptions{Scale: 2, QuietZone: 1}); err != nil {
		t.Fatalf("EncodePNG: %v", err)
	}
	img, err := png.Decode(&buf)
	if err != nil {
		t.Fatalf("png.Decode: %v", err)
	}
	want := (177 + 2) * 2
	if img.Bounds().Dx() != want {
		t.Errorf("size = %d, want %d", img.Bounds().Dx(), want)
	}
}

func TestEncodePNG_QuietZoneIsLight(t *testing.T) {
	res, err := engine.Synthesize(engine.Options{URL: "x"})
	if err != nil {
		t.Fatalf("Synthesize: %v", err)
	}
	var buf bytes.Buffer
	if err := res.EncodePNG(&buf, engine.PNGOptions{Scale: 1, QuietZone: 4}); err != nil {
		t.Fatalf("EncodePNG: %v", err)
	}
	img, err := png.Decode(&buf)
	if err != nil {
		t.Fatalf("png.Decode: %v", err)
	}
	// Top-left quiet-zone pixel must be light (gray 0xFF).
	rr, gg, bb, _ := img.At(0, 0).RGBA()
	if rr < 0xF000 || gg < 0xF000 || bb < 0xF000 {
		t.Errorf("quiet zone pixel = (%x,%x,%x), want light", rr, gg, bb)
	}
}
