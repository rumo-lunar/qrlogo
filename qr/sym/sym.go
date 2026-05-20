// Package sym represents bytes whose individual bits are linear
// forms over a fixed set of GF(2) free variables. These are the
// "ghost" values placed in the QR matrix during Phase 2 of the
// qrlogo pipeline, before the constraint solver collapses them to
// concrete bits.
//
// A Bit is `(XOR of selected variables) ⊕ Const`. A Byte is eight
// Bits in MSB-first order, mirroring how QR data codewords are
// packed in the symbol.
//
// All Bits originating from one Domain share the same variable
// indexing and the same backing-slice length.
package sym

import (
	"math/bits"

	"github.com/rumo-lunar/qrlogo/qr/gf256"
)

// Domain is a fixed universe of GF(2) free variables. Operations
// always preserve the Domain's Words length on their outputs.
type Domain struct {
	NumVars int
	Words   int
}

// NewDomain returns a Domain over numVars free variables.
func NewDomain(numVars int) *Domain {
	if numVars < 0 {
		panic("sym: negative numVars")
	}
	return &Domain{
		NumVars: numVars,
		Words:   (numVars + 63) / 64,
	}
}

// Bit is one linear form over the Domain's variables, plus a
// constant. Once a solution vector s is known, its value is
//
//	(XOR over i where Vars[i/64] bit (i%64) is set of s[i]) ⊕ Const
//
// Vars uses LSB-first packing within each uint64, matching the
// convention used by the bitset package.
type Bit struct {
	Vars  []uint64
	Const byte // 0 or 1
}

// Byte is eight Bits in MSB-first order: Bits[0] is the most
// significant bit of the represented byte.
type Byte [8]Bit

// ZeroBit returns the constant 0 bit (no variables).
func (d *Domain) ZeroBit() Bit {
	return Bit{Vars: make([]uint64, d.Words)}
}

// ConstBit returns the constant bit b (only its low bit is used).
func (d *Domain) ConstBit(b byte) Bit {
	return Bit{Vars: make([]uint64, d.Words), Const: b & 1}
}

// Variable returns the Bit equal to a single free variable.
func (d *Domain) Variable(idx int) Bit {
	if idx < 0 || idx >= d.NumVars {
		panic("sym: variable index out of range")
	}
	v := make([]uint64, d.Words)
	v[idx/64] = 1 << uint(idx%64)
	return Bit{Vars: v}
}

// XorBit returns a ⊕ b.
func (d *Domain) XorBit(a, b Bit) Bit {
	out := Bit{
		Vars:  make([]uint64, d.Words),
		Const: (a.Const ^ b.Const) & 1,
	}
	for i := 0; i < d.Words; i++ {
		out.Vars[i] = a.Vars[i] ^ b.Vars[i]
	}
	return out
}

// ConstByte returns the constant byte value v as a Byte with no
// variables.
func (d *Domain) ConstByte(v byte) Byte {
	var out Byte
	for i := 0; i < 8; i++ {
		out[i] = d.ConstBit((v >> uint(7-i)) & 1)
	}
	return out
}

// XorByte returns a ⊕ b, bit by bit.
func (d *Domain) XorByte(a, b Byte) Byte {
	var out Byte
	for i := 0; i < 8; i++ {
		out[i] = d.XorBit(a[i], b[i])
	}
	return out
}

// MulConst returns c · s in GF(256), where c is a concrete byte and
// s is symbolic.
//
// Multiplication by c is a linear map M_c: GF(2)^8 → GF(2)^8 over
// GF(2). We materialise M_c by multiplying c by each MSB-ordered
// basis vector 2^(7-i); the resulting bytes form a per-input-bit
// dependency mask. Each output Bit at MSB-position j is the XOR of
// input Bits at positions i for which bit (7-j) of M_c[i] is set.
func (d *Domain) MulConst(s Byte, c byte) Byte {
	if c == 0 {
		return d.ConstByte(0)
	}
	var matrix [8]byte
	for i := 0; i < 8; i++ {
		matrix[i] = gf256.Mul(c, 1<<uint(7-i))
	}
	var out Byte
	for j := 0; j < 8; j++ {
		outMask := byte(1) << uint(7-j)
		bit := d.ZeroBit()
		for i := 0; i < 8; i++ {
			if matrix[i]&outMask != 0 {
				bit = d.XorBit(bit, s[i])
			}
		}
		out[j] = bit
	}
	return out
}

// ResolveBit computes the concrete value of b given a packed
// solution vector. The solution layout matches bitset.Solve's
// output: MSB-first per byte, so variable idx n has value
// (solution[n/8] >> uint(7-n%8)) & 1.
func (d *Domain) ResolveBit(b Bit, solution []byte) byte {
	acc := b.Const & 1
	for w := 0; w < d.Words; w++ {
		word := b.Vars[w]
		for word != 0 {
			tz := bits.TrailingZeros64(word)
			varIdx := w*64 + tz
			if varIdx < d.NumVars {
				acc ^= (solution[varIdx/8] >> uint(7-varIdx%8)) & 1
			}
			word &= word - 1
		}
	}
	return acc & 1
}

// ResolveByte computes the concrete byte value of s given a solution.
func (d *Domain) ResolveByte(s Byte, solution []byte) byte {
	var v byte
	for i := 0; i < 8; i++ {
		v |= d.ResolveBit(s[i], solution) << uint(7-i)
	}
	return v
}
