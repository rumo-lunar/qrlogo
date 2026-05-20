package bitset_test

import (
	"testing"

	"github.com/rumo-lunar/qrlogo/bitset"
)

// setVars returns a packed []uint64 bitset of the given variable count,
// with bit indices in `indices` set to 1.
//
// Convention matches bitset.Row.Vars: LSB-first within each uint64,
// variable x_n at bit (n mod 64) of word n/64.
func setVars(numVars int, indices ...int) []uint64 {
	words := (numVars + 63) / 64
	v := make([]uint64, words)
	for _, i := range indices {
		v[i/64] |= 1 << uint(i%64)
	}
	return v
}

func TestSolve_Trivial1x1(t *testing.T) {
	// Arrange: x₀ = 1
	sut := &bitset.System{
		NumVars: 1,
		Rows: []bitset.Row{
			{Vars: setVars(1, 0), Target: 1},
		},
	}

	// Act
	bits, _, ok := sut.Solve()

	// Assert
	if !ok {
		t.Fatalf("expected consistent system, got inconsistent")
	}
	if len(bits) != 1 {
		t.Fatalf("expected 1 output byte, got %d", len(bits))
	}
	// x₀ = 1 → bit 0 (MSB of byte 0) = 1 → 0b10000000
	if bits[0] != 0b10000000 {
		t.Fatalf("expected bits[0]=0b10000000, got 0b%08b", bits[0])
	}
}

func TestSolve_Known3x3(t *testing.T) {
	// Arrange:
	//   x₀ ⊕ x₁       = 1
	//        x₁ ⊕ x₂  = 0
	//   x₀       ⊕ x₂ = 1
	//
	// Hand-computed RREF gives x₀=1, x₁=0, x₂=0 (column 2 is free → 0).
	sut := &bitset.System{
		NumVars: 3,
		Rows: []bitset.Row{
			{Vars: setVars(3, 0, 1), Target: 1},
			{Vars: setVars(3, 1, 2), Target: 0},
			{Vars: setVars(3, 0, 2), Target: 1},
		},
	}

	// Act
	bits, _, ok := sut.Solve()

	// Assert
	if !ok {
		t.Fatalf("expected consistent system, got inconsistent")
	}
	if len(bits) != 1 {
		t.Fatalf("expected 1 output byte, got %d", len(bits))
	}
	// x₀=1, x₁=0, x₂=0 → MSB-first byte: 1 0 0 0 0 0 0 0
	if bits[0] != 0b10000000 {
		t.Fatalf("expected bits[0]=0b10000000, got 0b%08b", bits[0])
	}
}

func TestSolve_Underdetermined2x3(t *testing.T) {
	// Arrange:
	//   x₀ ⊕ x₁ = 1
	//   x₀ ⊕ x₂ = 0
	//
	// Two equations in three unknowns. After Gauss-Jordan, columns 0
	// and 1 are pivots; column 2 is free → x₂ = 0.
	// That gives x₀ = 0, x₁ = 1, x₂ = 0.
	sut := &bitset.System{
		NumVars: 3,
		Rows: []bitset.Row{
			{Vars: setVars(3, 0, 1), Target: 1},
			{Vars: setVars(3, 0, 2), Target: 0},
		},
	}

	// Act
	bits, _, ok := sut.Solve()

	// Assert
	if !ok {
		t.Fatalf("expected consistent system, got inconsistent")
	}
	if len(bits) != 1 {
		t.Fatalf("expected 1 output byte, got %d", len(bits))
	}
	// x₀=0, x₁=1, x₂=0 → MSB-first byte: 0 1 0 0 0 0 0 0
	if bits[0] != 0b01000000 {
		t.Fatalf("expected bits[0]=0b01000000, got 0b%08b", bits[0])
	}

	// Cross-check: the returned solution must satisfy every input row.
	x0 := (bits[0] >> 7) & 1
	x1 := (bits[0] >> 6) & 1
	x2 := (bits[0] >> 5) & 1
	if x0^x1 != 1 {
		t.Errorf("equation 1 not satisfied: x₀⊕x₁=%d, want 1", x0^x1)
	}
	if x0^x2 != 0 {
		t.Errorf("equation 2 not satisfied: x₀⊕x₂=%d, want 0", x0^x2)
	}
}

func TestSolve_Inconsistent(t *testing.T) {
	// Arrange:
	//   x₀ = 0
	//   x₀ = 1
	//
	// After Gauss-Jordan, row 0 is the pivot and row 1 reduces to
	// 0 = 1, the contradiction. The conflict is at the original
	// row index 1.
	sut := &bitset.System{
		NumVars: 1,
		Rows: []bitset.Row{
			{Vars: setVars(1, 0), Target: 0},
			{Vars: setVars(1, 0), Target: 1},
		},
	}

	// Act
	_, conflictRow, ok := sut.Solve()

	// Assert
	if ok {
		t.Fatalf("expected inconsistent system, got ok")
	}
	if conflictRow != 1 {
		t.Errorf("expected conflictRow=1, got %d", conflictRow)
	}
}

func TestSolve_IdentityStress128(t *testing.T) {
	// Arrange: 128 equations, each pinning one variable: x_i = i mod 2.
	const n = 128
	rows := make([]bitset.Row, n)
	for i := 0; i < n; i++ {
		rows[i] = bitset.Row{
			Vars:   setVars(n, i),
			Target: byte(i % 2),
		}
	}
	sut := &bitset.System{
		NumVars: n,
		Rows:    rows,
	}

	// Act
	bits, _, ok := sut.Solve()

	// Assert
	if !ok {
		t.Fatalf("expected consistent system, got inconsistent")
	}
	if len(bits) != n/8 {
		t.Fatalf("expected %d output bytes, got %d", n/8, len(bits))
	}
	// Each byte holds x_{8k}..x_{8k+7} MSB-first:
	//   bit pattern 0 1 0 1 0 1 0 1 = 0x55
	for i, b := range bits {
		if b != 0x55 {
			t.Errorf("bits[%d] = 0x%02x, want 0x55", i, b)
		}
	}
}

// TestSolve_DoesNotMutate proves the contract that Solve operates on
// an internal copy of the system. Without this guarantee, /engine
// cannot retry solves with dropped constraints from the same System.
func TestSolve_DoesNotMutate(t *testing.T) {
	// Arrange
	originalVars := setVars(3, 0, 1)
	originalTarget := byte(1)
	sut := &bitset.System{
		NumVars: 3,
		Rows: []bitset.Row{
			{Vars: originalVars, Target: originalTarget},
			{Vars: setVars(3, 1, 2), Target: 0},
		},
	}

	// Act
	_, _, _ = sut.Solve()

	// Assert: first row's Vars and Target are unchanged.
	if sut.Rows[0].Target != originalTarget {
		t.Errorf("row 0 Target mutated: got %d, want %d", sut.Rows[0].Target, originalTarget)
	}
	if len(sut.Rows[0].Vars) != len(originalVars) {
		t.Fatalf("row 0 Vars length mutated")
	}
	for i, w := range originalVars {
		if sut.Rows[0].Vars[i] != w {
			t.Errorf("row 0 Vars[%d] mutated: got 0x%016x, want 0x%016x",
				i, sut.Rows[0].Vars[i], w)
		}
	}
}
