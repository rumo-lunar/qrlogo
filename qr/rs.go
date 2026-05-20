package qr

import (
	"github.com/rumo-lunar/qrlogo/qr/gf256"
	"github.com/rumo-lunar/qrlogo/qr/sym"
)

// EncodeRS computes numEC Reed–Solomon error-correction codewords
// over the given symbolic data codewords.
//
// Mathematically, it returns the coefficients of
//
//	(data(x) · x^numEC) mod g(x)
//
// where g(x) is the GF(256) generator polynomial of degree numEC
// produced by gf256.GeneratorPoly. The returned slice has length
// numEC; the first element is the highest-degree coefficient.
//
// Because every operation used is either symbolic XOR or
// multiplication by a concrete GF(256) constant (both of which are
// linear over GF(2)), every output Bit ends up as a linear form in
// the free variables of d plus a fixed constant offset contributed
// by the URL bits. This linearity is the property /engine relies on
// when pinning image pixels against ghost-grid modules.
//
// Panics if numEC <= 0 or len(data) == 0.
func EncodeRS(d *sym.Domain, data []sym.Byte, numEC int) []sym.Byte {
	if numEC <= 0 {
		panic("qr.EncodeRS: numEC must be positive")
	}
	if len(data) == 0 {
		panic("qr.EncodeRS: data must be non-empty")
	}

	g := gf256.GeneratorPoly(numEC) // length numEC+1, leading coeff = 1

	// Working buffer: data followed by numEC zero codewords. This is
	// the polynomial data(x) · x^numEC.
	work := make([]sym.Byte, len(data)+numEC)
	copy(work, data)
	zero := d.ConstByte(0)
	for i := len(data); i < len(work); i++ {
		work[i] = zero
	}

	// Polynomial long division. At step i, work[i] is the leading
	// symbolic coefficient; we subtract lead · g(x) · x^(deg-i) to
	// cancel it, then advance. Subtraction in GF(256) is XOR.
	for i := 0; i < len(data); i++ {
		lead := work[i]
		for j := 0; j <= numEC; j++ {
			work[i+j] = d.XorByte(work[i+j], d.MulConst(lead, g[j]))
		}
	}

	// The remainder occupies the last numEC slots.
	return work[len(data):]
}
