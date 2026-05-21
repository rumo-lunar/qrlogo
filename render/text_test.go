package render_test

import (
	"testing"

	"github.com/rumo-lunar/qrlogo/render"
)

func TestRenderText_ProducesOnlyBlackAndDontCare(t *testing.T) {
	tm := render.RenderText("HI", 61, 61)

	for i, p := range tm.Pixels {
		if p != render.PixelBlack && p != render.PixelDontCare {
			t.Fatalf("pixel %d = %v, want Black or DontCare", i, p)
		}
	}

	black, white, _ := tm.Counts()
	if black == 0 {
		t.Error("no black pixels — text was not rendered")
	}
	if white != 0 {
		t.Errorf("got %d white pixels, want 0 (halo not yet applied)", white)
	}
}

func TestRenderText_EmptyStringHasNoBlacks(t *testing.T) {
	tm := render.RenderText("", 20, 20)
	black, _, _ := tm.Counts()
	if black != 0 {
		t.Errorf("empty string produced %d black pixels", black)
	}
}

func TestRenderText_HaloMakesWhiteRingAroundGlyphs(t *testing.T) {
	tm := render.RenderText("A", 20, 20)
	render.ApplyHalo(tm)

	black, white, _ := tm.Counts()
	if black == 0 {
		t.Fatal("no black after render+halo")
	}
	if white == 0 {
		t.Error("no white halo created around glyphs")
	}
}

func TestRenderText_OutputDimensionsMatch(t *testing.T) {
	tm := render.RenderText("x", 40, 25)
	if tm.W != 40 || tm.H != 25 {
		t.Errorf("size = %dx%d, want 40x25", tm.W, tm.H)
	}
	if len(tm.Pixels) != 40*25 {
		t.Errorf("pixel count = %d, want %d", len(tm.Pixels), 40*25)
	}
}
