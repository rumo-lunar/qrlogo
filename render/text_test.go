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

// TestRenderText_FillsCenter checks that rendered text is centred in the
// target and large enough to occupy a substantial fraction of the grid —
// not pinned to the left edge in a tiny 7×13-pixel strip.
func TestRenderText_FillsCenter(t *testing.T) {
	// Arrange
	const w, h = 177, 177
	sut := render.RenderText

	// Act
	tm := sut("HI", w, h)

	// Assert: centre cell must be inside the text's bounding box.
	black, _, _ := tm.Counts()
	if black == 0 {
		t.Fatal("no black pixels rendered")
	}

	minR, minC, maxR, maxC := boundingBox(tm)

	// Bounding box must be roughly centred (left margin ≈ right margin,
	// top ≈ bottom — tolerate a few cells of asymmetry).
	leftMargin := minC
	rightMargin := w - 1 - maxC
	topMargin := minR
	bottomMargin := h - 1 - maxR

	const tolerance = 3
	if abs(leftMargin-rightMargin) > tolerance {
		t.Errorf("not horizontally centred: left=%d right=%d", leftMargin, rightMargin)
	}
	if abs(topMargin-bottomMargin) > tolerance {
		t.Errorf("not vertically centred: top=%d bottom=%d", topMargin, bottomMargin)
	}

	// Text must fill a meaningful fraction of the grid (≥ 50% in each axis).
	bbW := maxC - minC + 1
	bbH := maxR - minR + 1
	if bbW*2 < w {
		t.Errorf("text width %d is < 50%% of grid width %d", bbW, w)
	}
	if bbH*2 < h {
		t.Errorf("text height %d is < 50%% of grid height %d", bbH, h)
	}
}

func boundingBox(tm *render.TargetMap) (minR, minC, maxR, maxC int) {
	minR, minC = tm.H, tm.W
	maxR, maxC = -1, -1
	for r := 0; r < tm.H; r++ {
		for c := 0; c < tm.W; c++ {
			if tm.At(r, c) == render.PixelBlack {
				if r < minR {
					minR = r
				}
				if r > maxR {
					maxR = r
				}
				if c < minC {
					minC = c
				}
				if c > maxC {
					maxC = c
				}
			}
		}
	}
	return
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}
