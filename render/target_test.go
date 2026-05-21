package render_test

import (
	"testing"

	"github.com/rumo-lunar/qrlogo/render"
)

func TestNew_AllDontCare(t *testing.T) {
	tm := render.New(4, 3)
	if tm.W != 4 || tm.H != 3 {
		t.Fatalf("size = %dx%d, want 4x3", tm.W, tm.H)
	}
	if got := len(tm.Pixels); got != 12 {
		t.Fatalf("pixel count = %d, want 12", got)
	}
	for i, p := range tm.Pixels {
		if p != render.PixelDontCare {
			t.Errorf("pixel %d = %v, want DontCare", i, p)
		}
	}
}

func TestNew_PanicsOnNonPositiveSize(t *testing.T) {
	cases := [][2]int{{0, 1}, {1, 0}, {-1, 1}, {1, -1}}
	for _, c := range cases {
		func() {
			defer func() {
				if recover() == nil {
					t.Errorf("New(%d,%d) did not panic", c[0], c[1])
				}
			}()
			render.New(c[0], c[1])
		}()
	}
}

func TestSetAt_RoundTrip(t *testing.T) {
	tm := render.New(3, 2)
	tm.Set(0, 1, render.PixelBlack)
	tm.Set(1, 2, render.PixelWhite)

	if got := tm.At(0, 1); got != render.PixelBlack {
		t.Errorf("At(0,1) = %v, want Black", got)
	}
	if got := tm.At(1, 2); got != render.PixelWhite {
		t.Errorf("At(1,2) = %v, want White", got)
	}
	if got := tm.At(0, 0); got != render.PixelDontCare {
		t.Errorf("At(0,0) = %v, want DontCare", got)
	}
}

func TestAt_PanicsOutOfBounds(t *testing.T) {
	tm := render.New(2, 2)
	cases := [][2]int{{-1, 0}, {0, -1}, {2, 0}, {0, 2}}
	for _, c := range cases {
		func() {
			defer func() {
				if recover() == nil {
					t.Errorf("At(%d,%d) did not panic", c[0], c[1])
				}
			}()
			_ = tm.At(c[0], c[1])
		}()
	}
}

func TestForEachConstraint_VisitsOnlyBlackAndWhite(t *testing.T) {
	tm := render.New(3, 2)
	tm.Set(0, 0, render.PixelBlack)
	tm.Set(0, 1, render.PixelBlack)
	tm.Set(1, 0, render.PixelWhite)
	// (0,2), (1,1), (1,2) remain DontCare and must be skipped.

	type visit struct {
		r, c int
		bit  byte
	}
	var got []visit
	tm.ForEachConstraint(func(r, c int, wantBit byte) {
		got = append(got, visit{r, c, wantBit})
	})

	want := []visit{
		{0, 0, 1},
		{0, 1, 1},
		{1, 0, 0},
	}
	if len(got) != len(want) {
		t.Fatalf("got %d visits, want %d (%v)", len(got), len(want), got)
	}
	for i, w := range want {
		if got[i] != w {
			t.Errorf("visit %d = %+v, want %+v", i, got[i], w)
		}
	}
}

func TestCounts(t *testing.T) {
	tm := render.New(3, 2)
	tm.Set(0, 0, render.PixelBlack)
	tm.Set(0, 1, render.PixelBlack)
	tm.Set(1, 0, render.PixelWhite)

	b, w, d := tm.Counts()
	if b != 2 || w != 1 || d != 3 {
		t.Errorf("counts = (%d,%d,%d), want (2,1,3)", b, w, d)
	}
}

func TestInside(t *testing.T) {
	tm := render.New(3, 2)
	cases := []struct {
		r, c int
		want bool
	}{
		{0, 0, true}, {1, 2, true}, {-1, 0, false},
		{0, -1, false}, {2, 0, false}, {0, 3, false},
	}
	for _, tc := range cases {
		if got := tm.Inside(tc.r, tc.c); got != tc.want {
			t.Errorf("Inside(%d,%d) = %v, want %v", tc.r, tc.c, got, tc.want)
		}
	}
}
