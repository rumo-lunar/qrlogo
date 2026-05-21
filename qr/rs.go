// Package qr is the byte-level QR Code encoder. It exposes the
// per-version, per-EC-level pipeline (data → padding → RS → interleave
// → place → mask) on top of the spec tables in qr/spec.
//
// The package operates on plain []byte / [][]byte values; the symbolic
// QArt machinery that used to live here has been removed.
package qr

import "github.com/rumo-lunar/qrlogo/qr/gf256"

// EncodeRS computes numEC Reed–Solomon error-correction codewords
// over the given data codewords. It returns the coefficients of
//
//	(data(x) · x^numEC) mod g(x)
//
// where g(x) is the GF(256) generator polynomial of degree numEC
// produced by gf256.GeneratorPoly. The returned slice has length
// numEC; the first element is the highest-degree coefficient.
//
// Panics if numEC <= 0 or len(data) == 0.
func EncodeRS(data []byte, numEC int) []byte {
	if numEC <= 0 {
		panic("qr.EncodeRS: numEC must be positive")
	}
	if len(data) == 0 {
		panic("qr.EncodeRS: data must be non-empty")
	}

	g := gf256.GeneratorPoly(numEC) // length numEC+1, leading coeff = 1

	// Working buffer: data followed by numEC zero codewords. This is
	// the polynomial data(x) · x^numEC.
	work := make([]byte, len(data)+numEC)
	copy(work, data)

	// Synthetic polynomial long division. At step i, work[i] is the
	// leading coefficient; subtract lead · g(x) · x^(deg-i) to cancel
	// it, then advance. Subtraction in GF(256) is XOR.
	for i := 0; i < len(data); i++ {
		lead := work[i]
		if lead == 0 {
			continue
		}
		for j := 0; j <= numEC; j++ {
			work[i+j] ^= gf256.Mul(lead, g[j])
		}
	}

	// The remainder occupies the last numEC slots.
	return work[len(data):]
}
