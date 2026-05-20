package main

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"os"
	"strings"
	"testing"
)

// --- arg-parsing tests ---

func TestRun_MissingURL(t *testing.T) {
	var stderr bytes.Buffer
	err := run([]string{}, &bytes.Buffer{}, &stderr)
	assertExitCode(t, err, 1)
}

func TestRun_BothImageAndText(t *testing.T) {
	var stderr bytes.Buffer
	err := run([]string{"-url", "https://example.com", "-image", "foo.png", "-text", "HI"}, &bytes.Buffer{}, &stderr)
	assertExitCode(t, err, 1)
}

func TestRun_OversizedURL(t *testing.T) {
	url := strings.Repeat("x", 101)
	var stderr bytes.Buffer
	err := run([]string{"-url", url}, &bytes.Buffer{}, &stderr)
	assertExitCode(t, err, 2)
}

func TestRun_MissingImageFile(t *testing.T) {
	var stderr bytes.Buffer
	err := run([]string{"-url", "https://example.com", "-image", "/nonexistent/path.png"}, &bytes.Buffer{}, &stderr)
	assertExitCode(t, err, 2)
}

// --- end-to-end: plain QR to stdout ---

func TestRun_PlainQR_Stdout(t *testing.T) {
	var stdout, stderr bytes.Buffer
	err := run([]string{"-url", "https://example.com", "-out", "-"}, &stdout, &stderr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	img, err := png.Decode(&stdout)
	if err != nil {
		t.Fatalf("output is not a valid PNG: %v", err)
	}

	// V11 symbol: 61 modules + 2*4 quiet zone = 69 modules × 8 px/module = 552 px
	want := (61 + 2*4) * 8
	b := img.Bounds()
	if b.Dx() != want || b.Dy() != want {
		t.Errorf("image size %dx%d, want %dx%d", b.Dx(), b.Dy(), want, want)
	}

	// Timing pattern: module row 6, cols 8..52 should alternate dark/light.
	// With quiet zone 4 and scale 8, pixel centre of module (r, c) is at
	// ((quiet+r)*scale + scale/2, (quiet+c)*scale + scale/2).
	scale, quiet := 8, 4
	y := (quiet+6)*scale + scale/2
	for modCol := 8; modCol <= 52; modCol++ {
		x := (quiet+modCol)*scale + scale/2
		r, _, _, _ := img.At(x, y).RGBA()
		isDark := r < 0x8000
		wantDark := (modCol % 2) == 0
		if isDark != wantDark {
			t.Errorf("timing module col=%d: isDark=%v want %v", modCol, isDark, wantDark)
		}
	}
}

// --- end-to-end: text target ---

func TestRun_TextTarget(t *testing.T) {
	out := t.TempDir() + "/out.png"

	var stderr bytes.Buffer
	err := run([]string{
		"-url", "https://example.com",
		"-text", "HI",
		"-out", out,
	}, &bytes.Buffer{}, &stderr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	f, err := os.Open(out)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	img, err := png.Decode(f)
	if err != nil {
		t.Fatalf("output is not a valid PNG: %v", err)
	}

	// Text is centred vertically in the 61×61 symbol; scan the centre band
	// (module rows 24..37) for at least one dark pixel in the left columns.
	scale, quiet := 8, 4
	foundDark := false
outer:
	for row := quiet + 24; row <= quiet+37; row++ {
		for col := quiet + 1; col <= quiet+15; col++ {
			x := col*scale + scale/2
			y := row*scale + scale/2
			r, _, _, _ := img.At(x, y).RGBA()
			if r < 0x8000 {
				foundDark = true
				break outer
			}
		}
	}
	if !foundDark {
		t.Error("expected at least one dark pixel in the text target region")
	}
}

// --- end-to-end: logo-scale centres the logo in a sub-region ---

func TestRun_LogoScale(t *testing.T) {
	// An all-black PNG at scale 1.0 makes synthesis fail (over-constrained).
	// At scale 0.3 it fits in ~18×18 modules — well within solver capacity.
	blackPNG := writeBlackPNG(t, 16, 16)

	var stdout bytes.Buffer
	err := run([]string{
		"-url", "https://example.com",
		"-image", blackPNG,
		"-logo-scale", "0.3",
		"-out", "-",
	}, &stdout, &bytes.Buffer{})
	if err != nil {
		t.Fatalf("unexpected error at logo-scale 0.3: %v", err)
	}

	img, err := png.Decode(&stdout)
	if err != nil {
		t.Fatalf("output is not a valid PNG: %v", err)
	}

	// Centre of the symbol should contain dark pixels from the black logo.
	scale, quiet := 8, 4
	centre := quiet + 30 // module 30 ≈ centre of 61-module grid
	x := centre*scale + scale/2
	y := centre*scale + scale/2
	r, _, _, _ := img.At(x, y).RGBA()
	if r >= 0x8000 {
		t.Error("expected a dark pixel at symbol centre for all-black logo")
	}
}

func TestRun_BestEffort(t *testing.T) {
	blackPNG := writeBlackPNG(t, 16, 16)

	var stdout bytes.Buffer
	err := run([]string{
		"-url", "https://example.com",
		"-image", blackPNG,
		"-logo-scale", "1.0",
		"-best-effort",
		"-out", "-",
	}, &stdout, &bytes.Buffer{})
	if err != nil {
		t.Fatalf("expected synthesis to succeed with -best-effort, got: %v", err)
	}

	if _, err := png.Decode(&stdout); err != nil {
		t.Fatalf("output is not a valid PNG: %v", err)
	}
}

func TestRun_LogoScale_InvalidRange(t *testing.T) {
	var stderr bytes.Buffer
	err := run([]string{"-url", "https://example.com", "-logo-scale", "0"}, &bytes.Buffer{}, &stderr)
	assertExitCode(t, err, 1)
}

// --- end-to-end: image target + stats ---

func TestRun_ImageTarget_Stats(t *testing.T) {
	blackPNG := writeBlackPNG(t, 16, 16)

	var stderr bytes.Buffer
	err := run([]string{
		"-url", "https://example.com",
		"-image", blackPNG,
		"-out", "-",
		"-stats",
	}, &bytes.Buffer{}, &stderr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	out := stderr.String()
	if !strings.Contains(out, "function conflicts") {
		t.Errorf("stats output missing 'function conflicts'; got:\n%s", out)
	}
	if strings.Contains(out, "function conflicts: 0") {
		t.Errorf("expected non-zero function conflicts for all-black image; got:\n%s", out)
	}
}

// --- helpers ---

func assertExitCode(t *testing.T, err error, want int) {
	t.Helper()
	if err == nil {
		t.Fatalf("expected exit code %d, got nil error", want)
	}
	e, ok := err.(*exitError)
	if !ok {
		t.Fatalf("expected *exitError, got %T: %v", err, err)
	}
	if e.code != want {
		t.Errorf("exit code = %d, want %d (msg: %s)", e.code, want, e.msg)
	}
}

// writeBlackPNG creates a temporary PNG with a solid black square centred on a
// transparent background. The transparent background ensures IgnoreTransparent
// maps those pixels to DontCare so only the black square adds constraints.
// The black square overlaps the QR finder patterns, guaranteeing
// FunctionConflicts > 0 without over-constraining the data modules.
func writeBlackPNG(t *testing.T, w, h int) string {
	t.Helper()
	path := t.TempDir() + "/logo.png"
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	img := image.NewNRGBA(image.Rect(0, 0, w, h))
	// Draw a solid black square in the top-left quadrant (overlaps finder).
	sq := w / 3
	for y := 0; y < sq; y++ {
		for x := 0; x < sq; x++ {
			img.SetNRGBA(x, y, color.NRGBA{R: 0, G: 0, B: 0, A: 255})
		}
	}
	if err := png.Encode(f, img); err != nil {
		t.Fatal(err)
	}
	return path
}
