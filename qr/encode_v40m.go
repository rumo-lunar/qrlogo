package qr

import (
	"github.com/rumo-lunar/qrlogo/qr/sym"
)

// V40-M structural constants.
//
// See ISO/IEC 18004 Annex D, Table 9 for the derivation.
// Version 40, EC level M, byte mode.
const (
	// DataCodewordsV40M is the total number of data codewords in a
	// V40-M QR symbol, organised across 49 RS blocks as
	//
	//   18 × 47  +  31 × 48  =  2334.
	DataCodewordsV40M = 2334

	// ECCodewordsV40M is the total number of error-correction
	// codewords, 49 blocks × 28 EC per block.
	ECCodewordsV40M = 1372

	// MaxURLBytesV40M is the URL byte-length budget for V40-M.
	// Together with the 24 bits of byte-mode framing (4 mode + 16
	// length + 4 terminator), this fixes the free-padding budget at
	//
	//   (2334 − len(url) − 3) × 8 free bits.
	MaxURLBytesV40M = 2331
)

// EncodeDataV40M builds the symbolic data-codeword sequence for a
// byte-mode encoding of url at V40-M.
//
// It returns:
//
//   - codewords: a slice of exactly DataCodewordsV40M sym.Byte values.
//     The first len(url)+3 codewords are fixed (mode indicator +
//     16-bit character count + url payload + 4-bit terminator). The
//     remaining DataCodewordsV40M − len(url) − 3 codewords are
//     symbolic padding; each carries 8 fresh free variables MSB-first.
//   - d: a freshly constructed sym.Domain whose NumVars equals
//     (DataCodewordsV40M − len(url) − 3) × 8.
//
// Byte-mode framing for V40 (version ≥ 10) uses a 16-bit length
// field. The total forced bit count is 4 + 16 + 8·N + 4 = 8·(N+3),
// always a multiple of 8, so no trailing zero-bit padding is needed.
//
// Panics if url is empty or longer than MaxURLBytesV40M bytes.
func EncodeDataV40M(url string) (codewords []sym.Byte, d *sym.Domain) {
	n := len(url)
	if n == 0 {
		panic("qr.EncodeDataV40M: empty URL")
	}
	if n > MaxURLBytesV40M {
		panic("qr.EncodeDataV40M: URL exceeds MaxURLBytesV40M")
	}

	paddingCodewords := DataCodewordsV40M - (n + 3)
	d = sym.NewDomain(paddingCodewords * 8)

	// Pack the forced section MSB-first into n+3 bytes.
	bw := newBitWriter(n + 3)
	bw.write(0b0100, 4)   // mode indicator: byte
	bw.write(uint(n), 16) // character count (V≥10 byte mode is 16-bit)
	for i := 0; i < n; i++ {
		bw.write(uint(url[i]), 8)
	}
	bw.write(0, 4) // terminator (0000)
	if bw.bitPos != 0 {
		// Invariant: 4 + 16 + 8n + 4 = 8(n+3) is always byte-aligned.
		panic("qr.EncodeDataV40M: bitstream not byte-aligned")
	}
	if len(bw.buf) != n+3 {
		panic("qr.EncodeDataV40M: unexpected forced-section length")
	}

	codewords = make([]sym.Byte, DataCodewordsV40M)
	for i := 0; i < n+3; i++ {
		codewords[i] = d.ConstByte(bw.buf[i])
	}
	for k := 0; k < paddingCodewords; k++ {
		var b sym.Byte
		for j := 0; j < 8; j++ {
			b[j] = d.Variable(k*8 + j)
		}
		codewords[n+3+k] = b
	}
	return codewords, d
}
