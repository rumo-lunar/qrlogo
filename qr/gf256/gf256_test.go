package gf256_test

import (
	"testing"

	"github.com/rumo-lunar/qrlogo/qr/gf256"
)

// refMul is an independent, table-free implementation of GF(256)
// multiplication: schoolbook polynomial multiply of two bytes, then
// reduce modulo p(x) = 0x11D. It is the oracle against which Mul is
// validated.
func refMul(a, b byte) byte {
	var r uint16
	for i := 0; i < 8; i++ {
		if b&(1<<i) != 0 {
			r ^= uint16(a) << i
		}
	}
	for i := 14; i >= 8; i-- {
		if r&(1<<i) != 0 {
			r ^= 0x11D << uint(i-8)
		}
	}
	return byte(r)
}

func TestAdd_IsXOR(t *testing.T) {
	// Arrange + Act + Assert: addition in GF(256) must equal XOR.
	for a := 0; a < 256; a++ {
		for b := 0; b < 256; b++ {
			if got := gf256.Add(byte(a), byte(b)); got != byte(a)^byte(b) {
				t.Fatalf("Add(%d,%d) = %d, want %d", a, b, got, byte(a)^byte(b))
			}
		}
	}
}

func TestMul_AgainstReference(t *testing.T) {
	// Arrange + Act + Assert: Mul over all 65 536 byte pairs must
	// match the carry-less reference multiplier.
	for a := 0; a < 256; a++ {
		for b := 0; b < 256; b++ {
			got := gf256.Mul(byte(a), byte(b))
			want := refMul(byte(a), byte(b))
			if got != want {
				t.Fatalf("Mul(%d,%d) = %d, want %d", a, b, got, want)
			}
		}
	}
}

func TestMul_Identity(t *testing.T) {
	for a := 0; a < 256; a++ {
		if got := gf256.Mul(byte(a), 1); got != byte(a) {
			t.Errorf("Mul(%d, 1) = %d, want %d", a, got, a)
		}
	}
}

func TestMul_Zero(t *testing.T) {
	for a := 0; a < 256; a++ {
		if got := gf256.Mul(byte(a), 0); got != 0 {
			t.Errorf("Mul(%d, 0) = %d, want 0", a, got)
		}
		if got := gf256.Mul(0, byte(a)); got != 0 {
			t.Errorf("Mul(0, %d) = %d, want 0", a, got)
		}
	}
}

func TestInverse_MultipliesToOne(t *testing.T) {
	for a := 1; a < 256; a++ {
		inv := gf256.Inverse(byte(a))
		if got := gf256.Mul(byte(a), inv); got != 1 {
			t.Errorf("a=%d * Inverse(a)=%d = %d, want 1", a, inv, got)
		}
	}
}

func TestInverse_OfZero_Panics(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("expected Inverse(0) to panic, did not")
		}
	}()
	_ = gf256.Inverse(0)
}

func TestPow_Basics(t *testing.T) {
	// Pow(a, 0) = 1 for all a (including 0 by convention).
	for a := 0; a < 256; a++ {
		if got := gf256.Pow(byte(a), 0); got != 1 {
			t.Errorf("Pow(%d, 0) = %d, want 1", a, got)
		}
	}

	// Pow(a, 1) = a.
	for a := 0; a < 256; a++ {
		if got := gf256.Pow(byte(a), 1); got != byte(a) {
			t.Errorf("Pow(%d, 1) = %d, want %d", a, got, a)
		}
	}

	// Pow(0, n) = 0 for any n > 0.
	for n := 1; n <= 10; n++ {
		if got := gf256.Pow(0, n); got != 0 {
			t.Errorf("Pow(0, %d) = %d, want 0", n, got)
		}
	}

	// Pow(a, 255) = 1 for a != 0 (Fermat: |GF(256)*| = 255).
	for a := 1; a < 256; a++ {
		if got := gf256.Pow(byte(a), 255); got != 1 {
			t.Errorf("Pow(%d, 255) = %d, want 1", a, got)
		}
	}
}

func TestExp_Cyclic(t *testing.T) {
	// α has order 255: Exp(0) = Exp(255) = Exp(510) = 1.
	if got := gf256.Exp(0); got != 1 {
		t.Errorf("Exp(0) = %d, want 1", got)
	}
	if got := gf256.Exp(255); got != 1 {
		t.Errorf("Exp(255) = %d, want 1", got)
	}
	if got := gf256.Exp(510); got != 1 {
		t.Errorf("Exp(510) = %d, want 1", got)
	}
	// Exp(i) == Exp(i + 255) for the duplicated table.
	for i := 0; i < 255; i++ {
		if gf256.Exp(i) != gf256.Exp(i+255) {
			t.Errorf("Exp(%d) != Exp(%d)", i, i+255)
		}
	}
	// Exp(1) = α = 2, Exp(8) = 29 (first overflow), Exp(9) = 58.
	if got := gf256.Exp(1); got != 2 {
		t.Errorf("Exp(1) = %d, want 2", got)
	}
	if got := gf256.Exp(8); got != 29 {
		t.Errorf("Exp(8) = %d, want 29", got)
	}
	if got := gf256.Exp(9); got != 58 {
		t.Errorf("Exp(9) = %d, want 58", got)
	}
}

func TestGeneratorPoly_Degree2_KnownAnswer(t *testing.T) {
	// Arrange + Act
	g := gf256.GeneratorPoly(2)

	// Assert: g_2(x) = (x − α^0)(x − α^1) = x² + 3x + 2.
	want := []byte{1, 3, 2}
	if len(g) != 3 {
		t.Fatalf("len(g_2) = %d, want 3", len(g))
	}
	for i, w := range want {
		if g[i] != w {
			t.Errorf("g_2[%d] = %d, want %d", i, g[i], w)
		}
	}
}

func TestGeneratorPoly_LeadingCoefficientOne(t *testing.T) {
	for _, n := range []int{1, 2, 7, 10, 13, 17, 22, 26, 28, 30} {
		g := gf256.GeneratorPoly(n)
		if len(g) != n+1 {
			t.Errorf("len(g_%d) = %d, want %d", n, len(g), n+1)
		}
		if g[0] != 1 {
			t.Errorf("g_%d[0] = %d, want 1 (leading coefficient)", n, g[0])
		}
	}
}

func TestGeneratorPoly_RootsAreAlphaPowers(t *testing.T) {
	// For each generator g_n, evaluating at α^i for i ∈ [0, n) must
	// yield 0. This is the defining property of g_n.
	for _, n := range []int{2, 7, 10, 13, 17, 22, 26, 28, 30} {
		g := gf256.GeneratorPoly(n)
		for i := 0; i < n; i++ {
			x := gf256.Exp(i)
			if got := gf256.EvalPoly(g, x); got != 0 {
				t.Errorf("g_%d(α^%d) = g(%d) = %d, want 0", n, i, x, got)
			}
		}
	}
}

func TestEvalPoly_HornerKnownValue(t *testing.T) {
	// Arrange: p(x) = x² + 3x + 2.
	p := []byte{1, 3, 2}

	// Assert: p(α^0) = 1 + 3 + 2 = 0 (XOR);
	//         p(α^1) = 4 + Mul(3,2) + 2 = 4 ^ 6 ^ 2 = 0;
	//         p(α^2) = 16 + Mul(3,4) + 2 = 16 ^ 12 ^ 2 = 30.
	cases := []struct {
		x    byte
		want byte
	}{
		{1, 0},
		{2, 0},
		{4, 30},
	}
	for _, c := range cases {
		if got := gf256.EvalPoly(p, c.x); got != c.want {
			t.Errorf("p(%d) = %d, want %d", c.x, got, c.want)
		}
	}
}
