package main

import (
	"bytes"
	"errors"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRun_RequiresURL(t *testing.T) {
	// Arrange
	var stdout, stderr bytes.Buffer

	// Act
	err := run([]string{"-out", filepath.Join(t.TempDir(), "x.png")}, &stdout, &stderr)

	// Assert
	if err == nil {
		t.Fatal("want error when -url missing, got nil")
	}
	var ee *exitError
	if !errors.As(err, &ee) || ee.code != exitInvalidArgs {
		t.Errorf("exit code = %v, want %d", ee, exitInvalidArgs)
	}
}

func TestRun_BasicProducesValidPNG(t *testing.T) {
	// Arrange
	out := filepath.Join(t.TempDir(), "qr.png")
	var stdout, stderr bytes.Buffer

	// Act
	if err := run([]string{"-url", "https://lunar.app", "-out", out}, &stdout, &stderr); err != nil {
		t.Fatalf("run: %v", err)
	}

	// Assert
	f, err := os.Open(out)
	if err != nil {
		t.Fatalf("opening output: %v", err)
	}
	defer f.Close()
	if _, err := png.Decode(f); err != nil {
		t.Fatalf("png.Decode: %v", err)
	}
}

func TestRun_VersionAndECProduceExpectedDimensions(t *testing.T) {
	// Arrange: V10 → 57 modules; scale 8 default; quiet 4 default.
	out := filepath.Join(t.TempDir(), "qr.png")
	var stdout, stderr bytes.Buffer
	wantSide := (57 + 8) * 8

	// Act
	err := run([]string{"-url", "x", "-version", "10", "-ec", "M", "-out", out},
		&stdout, &stderr)
	if err != nil {
		t.Fatalf("run: %v", err)
	}

	// Assert
	f, _ := os.Open(out)
	defer f.Close()
	img, err := png.Decode(f)
	if err != nil {
		t.Fatalf("png.Decode: %v", err)
	}
	if img.Bounds().Dx() != wantSide {
		t.Errorf("width = %d, want %d", img.Bounds().Dx(), wantSide)
	}
	if img.Bounds().Dy() != wantSide {
		t.Errorf("height = %d, want %d", img.Bounds().Dy(), wantSide)
	}
}

func TestRun_BadECErrors(t *testing.T) {
	var stdout, stderr bytes.Buffer
	err := run([]string{"-url", "x", "-ec", "Z"}, &stdout, &stderr)
	if err == nil {
		t.Fatal("want error for bad EC, got nil")
	}
	var ee *exitError
	if !errors.As(err, &ee) || ee.code != exitInvalidArgs {
		t.Errorf("exit code = %v, want %d", ee, exitInvalidArgs)
	}
}

func TestRun_VersionOutOfRangeErrors(t *testing.T) {
	var stdout, stderr bytes.Buffer
	err := run([]string{"-url", "x", "-version", "41"}, &stdout, &stderr)
	if err == nil {
		t.Fatal("want error for version 41, got nil")
	}
	var ee *exitError
	if !errors.As(err, &ee) || ee.code != exitInvalidArgs {
		t.Errorf("exit code = %v, want %d", ee, exitInvalidArgs)
	}
}

func TestRun_StdoutOutput(t *testing.T) {
	var stdout, stderr bytes.Buffer
	if err := run([]string{"-url", "x", "-out", "-"}, &stdout, &stderr); err != nil {
		t.Fatalf("run: %v", err)
	}
	if _, err := png.Decode(&stdout); err != nil {
		t.Fatalf("png.Decode from stdout: %v", err)
	}
}

func TestRun_OverdenseLogoCoverageWarns(t *testing.T) {
	// Arrange: write a tiny PNG to use as the logo, then ask for 30 % coverage.
	dir := t.TempDir()
	logoPath := filepath.Join(dir, "logo.png")
	writeTestPNG(t, logoPath)
	out := filepath.Join(dir, "qr.png")

	var stdout, stderr bytes.Buffer

	// Act
	err := run([]string{
		"-url", "x",
		"-image", logoPath,
		"-logo-coverage", "0.30",
		"-out", out,
	}, &stdout, &stderr)
	if err != nil {
		t.Fatalf("run: %v", err)
	}

	// Assert: stderr carries the warning.
	if !strings.Contains(stderr.String(), "warning") {
		t.Errorf("stderr missing warning, got: %q", stderr.String())
	}
}

func TestRun_RoundedFindersDefaultsOn(t *testing.T) {
	// Sanity: -rounded-finders=false should not error; both runs
	// should produce a valid PNG with identical dimensions.
	dir := t.TempDir()
	var stdout, stderr bytes.Buffer

	roundedOut := filepath.Join(dir, "rounded.png")
	if err := run([]string{"-url", "x", "-out", roundedOut}, &stdout, &stderr); err != nil {
		t.Fatalf("rounded: %v", err)
	}
	squareOut := filepath.Join(dir, "square.png")
	if err := run([]string{"-url", "x", "-rounded-finders=false", "-out", squareOut},
		&stdout, &stderr); err != nil {
		t.Fatalf("square: %v", err)
	}

	a := decodeSize(t, roundedOut)
	b := decodeSize(t, squareOut)
	if a != b {
		t.Errorf("rounded vs square sizes differ: %d vs %d", a, b)
	}
}

func decodeSize(t *testing.T, path string) int {
	t.Helper()
	f, err := os.Open(path)
	if err != nil {
		t.Fatalf("open %s: %v", path, err)
	}
	defer f.Close()
	img, err := png.Decode(f)
	if err != nil {
		t.Fatalf("decode %s: %v", path, err)
	}
	return img.Bounds().Dx()
}

// writeTestPNG creates a 2×2 red PNG at path. Enough to exercise the
// logo overlay path without committing a real asset to the repo.
func writeTestPNG(t *testing.T, path string) {
	t.Helper()
	f, err := os.Create(path)
	if err != nil {
		t.Fatalf("create %s: %v", path, err)
	}
	defer f.Close()
	// Minimal 2×2 red PNG, hand-encoded via image/png.
	const w, h = 2, 2
	img := makeRedImage(w, h)
	if err := png.Encode(f, img); err != nil {
		t.Fatalf("encode %s: %v", path, err)
	}
}
