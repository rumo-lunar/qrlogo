package render_test

import (
	"testing"

	"github.com/rumo-lunar/qrlogo/render"
)

// Helper: build a TargetMap from an ASCII picture using '.'/' '/'#'.
func mustParse(t *testing.T, lines []string) *render.TargetMap {
	t.Helper()
	if len(lines) == 0 {
		t.Fatal("mustParse: no lines")
	}
	w := len(lines[0])
	tm := render.New(w, len(lines))
	for r, line := range lines {
		if len(line) != w {
			t.Fatalf("mustParse: row %d width %d, want %d", r, len(line), w)
		}
		for c, ch := range line {
			switch ch {
			case '.':
				tm.Set(r, c, render.PixelDontCare)
			case ' ':
				tm.Set(r, c, render.PixelWhite)
			case '#':
				tm.Set(r, c, render.PixelBlack)
			default:
				t.Fatalf("mustParse: bad char %q at (%d,%d)", ch, r, c)
			}
		}
	}
	return tm
}

func TestApplyHalo_SingleBlackInCentre(t *testing.T) {
	in := mustParse(t, []string{
		".....",
		".....",
		"..#..",
		".....",
		".....",
	})
	want := mustParse(t, []string{
		".....",
		".   .",
		". # .",
		".   .",
		".....",
	})

	render.ApplyHalo(in)

	if in.String() != want.String() {
		t.Errorf("got:\n%swant:\n%s", in, want)
	}
}

func TestApplyHalo_BlackInCornerDoesNotOverflow(t *testing.T) {
	in := mustParse(t, []string{
		"#..",
		"...",
		"...",
	})
	want := mustParse(t, []string{
		"# .",
		"  .",
		"...",
	})

	render.ApplyHalo(in)

	if in.String() != want.String() {
		t.Errorf("got:\n%swant:\n%s", in, want)
	}
}

func TestApplyHalo_DoesNotOverwriteExistingWhite(t *testing.T) {
	in := mustParse(t, []string{
		".....",
		".. ..",
		"..#..",
		".....",
		".....",
	})
	want := mustParse(t, []string{
		".....",
		".   .",
		". # .",
		".   .",
		".....",
	})

	render.ApplyHalo(in)

	if in.String() != want.String() {
		t.Errorf("got:\n%swant:\n%s", in, want)
	}
}

func TestApplyHalo_DoesNotChainBlackToBlackThenHalo(t *testing.T) {
	// Halos should be exactly 1 cell wide even when blacks touch.
	in := mustParse(t, []string{
		".....",
		".....",
		".##..",
		".....",
		".....",
	})
	want := mustParse(t, []string{
		".....",
		"    .",
		" ## .",
		"    .",
		".....",
	})

	render.ApplyHalo(in)

	if in.String() != want.String() {
		t.Errorf("got:\n%swant:\n%s", in, want)
	}
}

func TestApplyHalo_NoBlackIsNoOp(t *testing.T) {
	in := mustParse(t, []string{
		".....",
		".....",
		".....",
	})
	render.ApplyHalo(in)
	for _, p := range in.Pixels {
		if p != render.PixelDontCare {
			t.Errorf("got %v, want DontCare", p)
		}
	}
}

func TestApplyHalo_NilIsSafe(t *testing.T) {
	render.ApplyHalo(nil) // must not panic
}
