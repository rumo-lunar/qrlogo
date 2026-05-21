// Package gf256 implements arithmetic in the finite field GF(256)
// used by QR-code Reed–Solomon error correction.
//
// The field is GF(2)[x] / p(x) with the primitive polynomial
//
//	p(x) = x⁸ + x⁴ + x³ + x² + 1   (0x11D)
//
// and primitive element α = 2 (i.e. x). Every nonzero element of
// the field equals α^i for some i ∈ [0, 254]; element 0 has no
// logarithm.
//
// All operations are O(1) table lookups. The exp table is duplicated
// to length 512 so Mul never needs a modulo step.
package gf256

// prim is the field's primitive polynomial, truncated to a byte.
// The high bit (representing x⁸) is implicit during reduction.
const prim = 0x1D // 0x11D & 0xFF

var (
	// expTbl[i] = α^i. Length 512; expTbl[i] == expTbl[i mod 255] for
	// i ≥ 255. The duplication lets Mul use Log[a]+Log[b] (max 508)
	// directly as an index without modular reduction.
	expTbl [512]byte

	// logTbl[v] = i such that α^i = v, for v ∈ [1, 255].
	// logTbl[0] is meaningless; callers must guard against it.
	logTbl [256]byte
)

func init() {
	x := byte(1)
	for i := 0; i < 255; i++ {
		expTbl[i] = x
		logTbl[x] = byte(i)
		// Multiply by α = 2: left-shift, then reduce modulo p(x) if
		// the high bit overflows.
		high := x & 0x80
		x <<= 1
		if high != 0 {
			x ^= prim
		}
	}
	// Mirror the cyclic exp table.
	for i := 255; i < 512; i++ {
		expTbl[i] = expTbl[i-255]
	}
}

// Add returns a + b in GF(256). Addition (and subtraction) is XOR.
func Add(a, b byte) byte { return a ^ b }

// Mul returns a · b in GF(256).
func Mul(a, b byte) byte {
	if a == 0 || b == 0 {
		return 0
	}
	return expTbl[int(logTbl[a])+int(logTbl[b])]
}

// Inverse returns a⁻¹ for a ≠ 0. Panics if a == 0.
func Inverse(a byte) byte {
	if a == 0 {
		panic("gf256: inverse of zero")
	}
	return expTbl[255-int(logTbl[a])]
}

// Pow returns a^n in GF(256) for any integer n (positive, zero, or
// negative). By convention 0^0 == 1 and 0^n == 0 for n > 0.
func Pow(a byte, n int) byte {
	if a == 0 {
		if n == 0 {
			return 1
		}
		return 0
	}
	e := (int(logTbl[a]) * n) % 255
	if e < 0 {
		e += 255
	}
	return expTbl[e]
}

// Exp returns α^i for any integer i.
func Exp(i int) byte {
	e := i % 255
	if e < 0 {
		e += 255
	}
	return expTbl[e]
}

// GeneratorPoly returns the Reed–Solomon generator polynomial of
// degree n,
//
//	g(x) = (x − α^0)(x − α^1) … (x − α^(n−1)).
//
// Coefficients are returned in descending-degree order: the result
// has length n+1, the leading coefficient (index 0) is always 1,
// and the constant term sits at index n.
//
// In GF(256), subtraction is XOR, so the factors are equivalently
// (x + α^i).
func GeneratorPoly(n int) []byte {
	poly := []byte{1}
	for i := 0; i < n; i++ {
		alphaI := expTbl[i]
		// Multiply poly (degree d) by (x + alphaI) → degree d+1.
		next := make([]byte, len(poly)+1)
		for j, c := range poly {
			next[j] ^= c                // c · x → next degree slot
			next[j+1] ^= Mul(c, alphaI) // c · α^i → same slot, shifted
		}
		poly = next
	}
	return poly
}

// EvalPoly evaluates polynomial p (descending-degree order) at x in
// GF(256) using Horner's method.
func EvalPoly(p []byte, x byte) byte {
	var y byte
	for _, c := range p {
		y = Add(Mul(y, x), c)
	}
	return y
}
