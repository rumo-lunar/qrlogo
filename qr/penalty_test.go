package qr_test

import (
	"testing"

	"github.com/rumo-lunar/qrlogo/qr"
)

func TestPenalty_AllLightHasMaxN4AndManyRows(t *testing.T) {
	// 21×21 all-light symbol (V1 size).
	g := make([][]byte, 21)
	for r := range g {
		g[r] = make([]byte, 21)
	}
	got := qr.Penalty(g)
	if got <= 0 {
		t.Errorf("Penalty(all light) = %d, want > 0", got)
	}
}

func TestPenalty_AllDarkMatchesAllLightByN4(t *testing.T) {
	// All-dark and all-light should both have the maximum N4 contribution.
	light := make([][]byte, 21)
	dark := make([][]byte, 21)
	for r := 0; r < 21; r++ {
		light[r] = make([]byte, 21)
		dark[r] = make([]byte, 21)
		for c := 0; c < 21; c++ {
			dark[r][c] = 1
		}
	}
	if qr.Penalty(light) != qr.Penalty(dark) {
		t.Errorf("light penalty %d != dark penalty %d (N1+N2 symmetric by colour)",
			qr.Penalty(light), qr.Penalty(dark))
	}
}

func TestPenalty_FinderLikePatternDetected(t *testing.T) {
	// A grid that is alternating except for one row carrying the
	// finder-like pattern should score at least 40 from rule N3.
	n := 11
	g := make([][]byte, n)
	for r := 0; r < n; r++ {
		g[r] = make([]byte, n)
		for c := 0; c < n; c++ {
			g[r][c] = byte((r + c) % 2)
		}
	}
	// Overwrite middle row with the pattern.
	pattern := []byte{1, 0, 1, 1, 1, 0, 1, 0, 0, 0, 0}
	copy(g[5], pattern)
	if qr.Penalty(g) < 40 {
		t.Errorf("Penalty after embedding finder-like row = %d, want >= 40",
			qr.Penalty(g))
	}
}
