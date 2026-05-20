package sym_test

import (
	"testing"

	"github.com/rumo-lunar/qrlogo/qr/gf256"
	"github.com/rumo-lunar/qrlogo/qr/sym"
)

func TestConstByte_ResolvesToConstantForAnyValue(t *testing.T) {
	// Arrange
	d := sym.NewDomain(1208)
	sol := make([]byte, 151) // ignored — ConstByte has no variables

	// Act + Assert: every byte value v round-trips through ConstByte.
	for v := 0; v < 256; v++ {
		if got := d.ResolveByte(d.ConstByte(byte(v)), sol); got != byte(v) {
			t.Errorf("ConstByte(%d) resolved to %d", v, got)
		}
	}
}

func TestVariable_PicksCorrectSolutionBit(t *testing.T) {
	// Arrange: solution byte 0x55 = 0b01010101 (MSB-first) → x_i = i mod 2.
	d := sym.NewDomain(128)
	sol := []byte{
		0x55, 0x55, 0x55, 0x55, 0x55, 0x55, 0x55, 0x55,
		0x55, 0x55, 0x55, 0x55, 0x55, 0x55, 0x55, 0x55,
	}

	// Act + Assert
	for i := 0; i < 128; i++ {
		want := byte(i % 2)
		if got := d.ResolveBit(d.Variable(i), sol); got != want {
			t.Errorf("Variable(%d) resolved to %d, want %d", i, got, want)
		}
	}
}

func TestXorBit_LinearOverSolution(t *testing.T) {
	// Arrange: sol[0] = 0b10110100 = 180
	//   x_0=1, x_1=0, x_2=1, x_3=1, x_4=0, x_5=1, x_6=0, x_7=0
	d := sym.NewDomain(10)
	sol := []byte{0b10110100, 0b00000000}

	x0 := d.Variable(0)
	x3 := d.Variable(3)
	x5 := d.Variable(5)

	// Act + Assert: x_0 ⊕ x_3 = 1 ⊕ 1 = 0
	if got := d.ResolveBit(d.XorBit(x0, x3), sol); got != 0 {
		t.Errorf("x_0 ⊕ x_3 = %d, want 0", got)
	}

	// x_0 ⊕ x_3 ⊕ x_5 = 1 ⊕ 1 ⊕ 1 = 1
	expr := d.XorBit(d.XorBit(x0, x3), x5)
	if got := d.ResolveBit(expr, sol); got != 1 {
		t.Errorf("x_0 ⊕ x_3 ⊕ x_5 = %d, want 1", got)
	}

	// x_0 ⊕ 1 = 1 ⊕ 1 = 0
	if got := d.ResolveBit(d.XorBit(x0, d.ConstBit(1)), sol); got != 0 {
		t.Errorf("x_0 ⊕ 1 = %d, want 0", got)
	}
}

func TestMulConst_ZeroIsConstZero(t *testing.T) {
	// Arrange
	d := sym.NewDomain(64)
	sol := []byte{0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF}
	var s sym.Byte
	for i := 0; i < 8; i++ {
		s[i] = d.Variable(i)
	}

	// Act
	out := d.MulConst(s, 0)

	// Assert: 0 · anything = 0
	if got := d.ResolveByte(out, sol); got != 0 {
		t.Errorf("MulConst(s, 0) resolved to %d, want 0", got)
	}
}

func TestMulConst_OneIsIdentity(t *testing.T) {
	// Arrange: sol[0] = 0xA3 = 0b10100011 → x_0..x_7 = 1,0,1,0,0,0,1,1
	d := sym.NewDomain(64)
	sol := []byte{0xA3, 0, 0, 0, 0, 0, 0, 0}
	var s sym.Byte
	for i := 0; i < 8; i++ {
		s[i] = d.Variable(i)
	}

	// Act
	out := d.MulConst(s, 1)

	// Assert
	if got := d.ResolveByte(out, sol); got != 0xA3 {
		t.Errorf("MulConst(s, 1) resolved to 0x%02x, want 0xA3", got)
	}
}

// TestMulConst_AgainstGFMul is the master correctness test for the
// linear map M_c. For every (v, c) pair, multiplying the constant
// symbolic byte v by c must agree with the concrete reference
// multiplication in GF(256).
func TestMulConst_AgainstGFMul(t *testing.T) {
	// Arrange
	d := sym.NewDomain(8) // no variables actually used
	sol := []byte{0}

	// Act + Assert over all 65 536 byte pairs.
	for v := 0; v < 256; v++ {
		sb := d.ConstByte(byte(v))
		for c := 0; c < 256; c++ {
			got := d.ResolveByte(d.MulConst(sb, byte(c)), sol)
			want := gf256.Mul(byte(c), byte(v))
			if got != want {
				t.Fatalf("MulConst(ConstByte(%d), %d) = %d, want %d",
					v, c, got, want)
			}
		}
	}
}

// TestMulConst_LinearOverXor verifies M_c(a ⊕ b) = M_c(a) ⊕ M_c(b)
// for symbolic a and b built from disjoint variable sets.
func TestMulConst_LinearOverXor(t *testing.T) {
	// Arrange: a holds x_0..x_7, b holds x_8..x_15. Solution gives
	// concrete values to all 16 variables.
	d := sym.NewDomain(16)
	sol := []byte{0xA5, 0x3C} // arbitrary

	var a, b sym.Byte
	for i := 0; i < 8; i++ {
		a[i] = d.Variable(i)
		b[i] = d.Variable(i + 8)
	}

	// Act + Assert across all constants c.
	for c := 0; c < 256; c++ {
		lhs := d.ResolveByte(d.MulConst(d.XorByte(a, b), byte(c)), sol)
		rhs := d.ResolveByte(
			d.XorByte(d.MulConst(a, byte(c)), d.MulConst(b, byte(c))),
			sol,
		)
		if lhs != rhs {
			t.Errorf("c=%d: M_c(a⊕b)=%d, M_c(a)⊕M_c(b)=%d", c, lhs, rhs)
		}
	}
}

// TestMulConst_EndToEnd_VariableBytes builds two symbolic bytes from
// variables, picks an arbitrary solution, and checks that the
// symbolic product equals the concrete GF(256) product of the
// resolved bytes. This is the most realistic preview of how /qr will
// use MulConst during Reed–Solomon encoding.
func TestMulConst_EndToEnd_VariableBytes(t *testing.T) {
	// Arrange
	d := sym.NewDomain(16)
	sol := []byte{0b11010010, 0b01101001}
	//   byte 0 → x_0..x_7  = 1,1,0,1,0,0,1,0 → value 0xD2
	//   byte 1 → x_8..x_15 = 0,1,1,0,1,0,0,1 → value 0x69

	var s sym.Byte
	for i := 0; i < 8; i++ {
		s[i] = d.Variable(i) // s resolves to 0xD2
	}

	// Act + Assert: c · s resolves to gf256.Mul(c, 0xD2) for every c.
	for c := 0; c < 256; c++ {
		got := d.ResolveByte(d.MulConst(s, byte(c)), sol)
		want := gf256.Mul(byte(c), 0xD2)
		if got != want {
			t.Fatalf("c=%d: MulConst symbolic = %d, want %d", c, got, want)
		}
	}
}
