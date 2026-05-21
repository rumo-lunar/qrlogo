package engine_test

import (
	"bytes"
	"image"
	"image/png"
	"testing"

	"github.com/rumo-lunar/qrlogo/engine"
	"github.com/rumo-lunar/qrlogo/qr/spec"
)

func TestEncode_AutoFitPicksSmallestVersionForShortURL(t *testing.T) {
	// Arrange
	opts := engine.Options{URL: "https://lunar.app", EC: spec.ECHigh}

	// Act
	res, err := engine.Encode(opts)
	if err != nil {
		t.Fatalf("Encode: %v", err)
	}

	// Assert
	if res.Spec.Version < 1 || res.Spec.Version > 5 {
		t.Errorf("Version = %d, expected a small version (≤5) for 17-byte URL at H",
			res.Spec.Version)
	}
	if res.Spec.EC != spec.ECHigh {
		t.Errorf("EC = %s, want H", res.Spec.EC)
	}
	wantSize := res.Spec.Version.Size()
	if len(res.Symbol) != wantSize {
		t.Errorf("Symbol size = %d, want %d", len(res.Symbol), wantSize)
	}
}

func TestEncode_DefaultsToECHigh(t *testing.T) {
	res, err := engine.Encode(engine.Options{URL: "x"})
	if err != nil {
		t.Fatalf("Encode: %v", err)
	}
	if res.Spec.EC != spec.ECHigh {
		t.Errorf("default EC = %s, want H", res.Spec.EC)
	}
}

func TestEncodePNG_DimensionsMatchScaleAndQuietZone(t *testing.T) {
	// Arrange
	res, err := engine.Encode(engine.Options{URL: "hello", EC: spec.ECMedium})
	if err != nil {
		t.Fatalf("Encode: %v", err)
	}
	var buf bytes.Buffer

	// Act
	err = res.EncodePNG(&buf, engine.PNGOptions{Scale: 10, QuietZone: 4})
	if err != nil {
		t.Fatalf("EncodePNG: %v", err)
	}

	// Assert
	img, err := png.Decode(&buf)
	if err != nil {
		t.Fatalf("png.Decode: %v", err)
	}
	wantSide := (res.Spec.Version.Size() + 2*4) * 10
	if got := img.Bounds().Dx(); got != wantSide {
		t.Errorf("PNG width = %d, want %d", got, wantSide)
	}
	if got := img.Bounds().Dy(); got != wantSide {
		t.Errorf("PNG height = %d, want %d", got, wantSide)
	}
}

func TestEncodePNG_DotModulesProducesValidPNG(t *testing.T) {
	// Arrange
	res, err := engine.Encode(engine.Options{URL: "https://lunar.app", EC: spec.ECHigh})
	if err != nil {
		t.Fatalf("Encode: %v", err)
	}
	var buf bytes.Buffer

	// Act
	if err := res.EncodePNG(&buf, engine.PNGOptions{
		ModuleShape: engine.ModuleShapeDot,
	}); err != nil {
		t.Fatalf("EncodePNG: %v", err)
	}

	// Assert: round-trips as a valid PNG of the expected dimensions.
	img, err := png.Decode(&buf)
	if err != nil {
		t.Fatalf("png.Decode: %v", err)
	}
	wantSide := (res.Spec.Version.Size() + 8) * 8
	if img.Bounds().Dx() != wantSide {
		t.Errorf("width = %d, want %d", img.Bounds().Dx(), wantSide)
	}
}

func TestParseModuleShape(t *testing.T) {
	cases := map[string]engine.ModuleShape{
		"square": engine.ModuleShapeSquare,
		"SQUARE": engine.ModuleShapeSquare,
		"dot":    engine.ModuleShapeDot,
		"DOT":    engine.ModuleShapeDot,
	}
	for in, want := range cases {
		got, err := engine.ParseModuleShape(in)
		if err != nil {
			t.Errorf("ParseModuleShape(%q): %v", in, err)
		}
		if got != want {
			t.Errorf("ParseModuleShape(%q) = %v, want %v", in, got, want)
		}
	}
	if _, err := engine.ParseModuleShape("hexagon"); err == nil {
		t.Errorf("ParseModuleShape(hexagon): want error, got nil")
	}
}

func TestEncodePNG_RejectsEmptySymbol(t *testing.T) {
	res := &engine.Result{}
	err := res.EncodePNG(&bytes.Buffer{}, engine.PNGOptions{})
	if err == nil {
		t.Fatal("EncodePNG: want error on empty symbol, got nil")
	}
}

func TestEncodePNG_RejectsBadLogoCoverage(t *testing.T) {
	res, _ := engine.Encode(engine.Options{URL: "x"})
	err := res.EncodePNG(&bytes.Buffer{}, engine.PNGOptions{
		Logo:         image.NewRGBA(image.Rect(0, 0, 1, 1)),
		LogoCoverage: 1.5,
	})
	if err == nil {
		t.Fatal("EncodePNG: want error on coverage > 1, got nil")
	}
}

func TestEncodePNG_WithNonSquareLogoStillSucceeds(t *testing.T) {
	// Arrange: a deliberately wide 200×60 logo. Should be scaled so
	// the longer side equals LogoCoverage * symbol width and the
	// shorter side scales proportionally — no panic, no distortion,
	// no validation error.
	res, err := engine.Encode(engine.Options{URL: "https://lunar.app", EC: spec.ECHigh})
	if err != nil {
		t.Fatalf("Encode: %v", err)
	}
	var buf bytes.Buffer

	// Act
	err = res.EncodePNG(&buf, engine.PNGOptions{
		Logo:         image.NewRGBA(image.Rect(0, 0, 200, 60)),
		LogoCoverage: 0.22,
		LogoPadding:  0.10,
	})
	if err != nil {
		t.Fatalf("EncodePNG: %v", err)
	}
	if _, err := png.Decode(&buf); err != nil {
		t.Fatalf("png.Decode: %v", err)
	}
}

func TestEncodePNG_WithLogoPaintsAndStaysSquare(t *testing.T) {
	// Arrange
	res, err := engine.Encode(engine.Options{URL: "https://lunar.app", EC: spec.ECHigh})
	if err != nil {
		t.Fatalf("Encode: %v", err)
	}
	var buf bytes.Buffer

	// Act
	err = res.EncodePNG(&buf, engine.PNGOptions{
		Logo:         image.NewRGBA(image.Rect(0, 0, 100, 100)),
		LogoCoverage: 0.2,
		LogoPadding:  0.1,
		Scale:        8,
		QuietZone:    4,
	})
	if err != nil {
		t.Fatalf("EncodePNG: %v", err)
	}

	// Assert
	img, err := png.Decode(&buf)
	if err != nil {
		t.Fatalf("png.Decode: %v", err)
	}
	if img.Bounds().Dx() != img.Bounds().Dy() {
		t.Errorf("PNG not square: %v", img.Bounds())
	}
}
