package render_test

import (
	"image"
	"image/color"
	"testing"

	"github.com/rumo-lunar/qrlogo/render"
)

// solid returns a w×h image filled with c.
func solid(w, h int, c color.Color) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for r := 0; r < h; r++ {
		for col := 0; col < w; col++ {
			img.Set(col, r, c)
		}
	}
	return img
}

func TestFromImage_AllBlackSourceProducesAllBlack(t *testing.T) {
	src := solid(10, 10, color.Black)
	tm := render.FromImage(src, 5, 5, render.ImageOptions{})

	for i, p := range tm.Pixels {
		if p != render.PixelBlack {
			t.Fatalf("pixel %d = %v, want Black", i, p)
		}
	}
}

func TestFromImage_AllWhiteSourceProducesAllDontCareByDefault(t *testing.T) {
	src := solid(10, 10, color.White)
	tm := render.FromImage(src, 5, 5, render.ImageOptions{})

	for i, p := range tm.Pixels {
		if p != render.PixelDontCare {
			t.Fatalf("pixel %d = %v, want DontCare", i, p)
		}
	}
}

func TestFromImage_OpaqueBackgroundMapsLightToWhite(t *testing.T) {
	src := solid(8, 8, color.White)
	tm := render.FromImage(src, 4, 4, render.ImageOptions{
		OpaqueBackground: true,
	})

	for i, p := range tm.Pixels {
		if p != render.PixelWhite {
			t.Fatalf("pixel %d = %v, want White", i, p)
		}
	}
}

func TestFromImage_IgnoreTransparentLeavesAlpha0AsDontCare(t *testing.T) {
	// Fully transparent black: would otherwise threshold to Black.
	src := solid(4, 4, color.RGBA{R: 0, G: 0, B: 0, A: 0})
	tm := render.FromImage(src, 4, 4, render.ImageOptions{
		IgnoreTransparent: true,
	})

	for i, p := range tm.Pixels {
		if p != render.PixelDontCare {
			t.Fatalf("pixel %d = %v, want DontCare (transparent)", i, p)
		}
	}
}

func TestFromImage_DimensionsMatch(t *testing.T) {
	src := solid(20, 20, color.Black)
	tm := render.FromImage(src, 7, 11, render.ImageOptions{})

	if tm.W != 7 || tm.H != 11 {
		t.Errorf("size = %dx%d, want 7x11", tm.W, tm.H)
	}
}

func TestFromImage_ThresholdCutsoffMidGrey(t *testing.T) {
	// Mid-grey (0x8080) is exactly at the default threshold; bump
	// slightly above and below to confirm direction.
	dark := solid(4, 4, color.RGBA{R: 0x40, G: 0x40, B: 0x40, A: 0xFF})
	light := solid(4, 4, color.RGBA{R: 0xC0, G: 0xC0, B: 0xC0, A: 0xFF})

	td := render.FromImage(dark, 2, 2, render.ImageOptions{})
	tl := render.FromImage(light, 2, 2, render.ImageOptions{})

	for i, p := range td.Pixels {
		if p != render.PixelBlack {
			t.Errorf("dark pixel %d = %v, want Black", i, p)
		}
	}
	for i, p := range tl.Pixels {
		if p != render.PixelDontCare {
			t.Errorf("light pixel %d = %v, want DontCare", i, p)
		}
	}
}
