package render_test

import (
	"strings"
	"testing"

	"github.com/rumo-lunar/qrlogo/render"
)

func TestRenderText_ProducesOnlyBlackAndDontCare(t *testing.T) {
	tm := render.RenderText("HI", 61, 61, render.TextOptions{})

	for i, p := range tm.Pixels {
		if p != render.PixelBlack && p != render.PixelDontCare {
			t.Fatalf("pixel %d = %v, want Black or DontCare", i, p)
		}
	}

	black, white, _ := tm.Counts()
	if black == 0 {
		t.Errorf("no black pixels — text was not rendered:\n%s", tm)
	}
	if white != 0 {
		t.Errorf("got %d white pixels, want 0 (halo not yet applied)", white)
	}
}

func TestRenderText_EmptyStringHasNoBlacks(t *testing.T) {
	tm := render.RenderText("", 20, 20, render.TextOptions{})
	black, _, _ := tm.Counts()
	if black != 0 {
		t.Errorf("empty string produced %d black pixels", black)
	}
}

func TestRenderText_HaloMakesWhiteRingAroundGlyphs(t *testing.T) {
	tm := render.RenderText("A", 20, 20, render.TextOptions{})
	render.ApplyHalo(tm)

	black, white, _ := tm.Counts()
	if black == 0 {
		t.Fatalf("no black after render+halo")
	}
	if white == 0 {
		t.Errorf("no white halo created around glyphs:\n%s", tm)
	}
}

func TestRenderText_OutputDimensionsMatch(t *testing.T) {
	tm := render.RenderText("x", 40, 25, render.TextOptions{})
	if tm.W != 40 || tm.H != 25 {
		t.Errorf("size = %dx%d, want 40x25", tm.W, tm.H)
	}
	if len(tm.Pixels) != 40*25 {
		t.Errorf("pixel count = %d, want %d", len(tm.Pixels), 40*25)
	}
}

func TestRenderText_DefaultStringStartsWithDontCare(t *testing.T) {
	// Smoke test the textual representation is sane: it must have
	// H rows separated by newlines and W chars per row.
	tm := render.RenderText("hi", 12, 6, render.TextOptions{})
	rows := strings.Split(strings.TrimRight(tm.String(), "\n"), "\n")
	if got := len(rows); got != 6 {
		t.Fatalf("got %d rows, want 6", got)
	}
	for r, row := range rows {
		if len(row) != 12 {
			t.Errorf("row %d width = %d, want 12", r, len(row))
		}
	}
}
