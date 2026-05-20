package qr

import (
	"github.com/rumo-lunar/qrlogo/qr/sym"
)

// V11-M structural constants.
//
// See README "Capacity budget" for the derivation. These are baked
// into the v1 contract; widening to other versions / EC levels is a
// v2 concern.
const (
	// DataCodewordsV11M is the total number of data codewords in a
	// V11-M QR symbol, organised across 5 RS blocks as
	//
	//   1 × 50  +  4 × 51  =  254.
	DataCodewordsV11M = 254

	// ECCodewordsV11M is the total number of error-correction
	// codewords, 5 blocks × 30 EC per block.
	ECCodewordsV11M = 150

	// MaxURLBytesV11M is the URL byte-length budget for v1's contract.
	// Together with the 24 bits of byte-mode framing (4 mode + 16
	// length + 4 terminator), this fixes the free-padding budget at
	//
	//   (254 − 100 − 3) × 8 = 1208 free bits.
	MaxURLBytesV11M = 100
)

// EncodeData builds the symbolic data-codeword sequence for a
// byte-mode encoding of url at V11-M.
//
// It returns:
//
//   - codewords: a slice of exactly DataCodewordsV11M sym.Byte values.
//     The first len(url)+3 codewords are fixed (mode indicator +
//     16-bit character count + url payload + 4-bit terminator). The
//     remaining DataCodewordsV11M − len(url) − 3 codewords are
//     symbolic padding; each carries 8 fresh free variables MSB-first.
//   - d: a freshly constructed sym.Domain whose NumVars equals
//     (DataCodewordsV11M − len(url) − 3) × 8.
//
// Variable numbering is sequential across padding codewords. The
// first padding codeword carries variables 0..7, the second 8..15,
// and so on. Variable n appears at MSB-position (n mod 8) of
// codewords[(n/8) + len(url) + 3].
//
// Byte-mode framing for V11 (version ≥ 10) uses a 16-bit length
// field. The total forced bit count is 4 + 16 + 8·N + 4 = 8·(N+3),
// always a multiple of 8, so no trailing zero-bit padding is needed.
//
// Panics if url is empty or longer than MaxURLBytesV11M bytes.
func EncodeData(url string) (codewords []sym.Byte, d *sym.Domain) {
	n := len(url)
	if n == 0 {
		panic("qr.EncodeData: empty URL")
	}
	if n > MaxURLBytesV11M {
		panic("qr.EncodeData: URL exceeds MaxURLBytesV11M")
	}

	paddingCodewords := DataCodewordsV11M - (n + 3)
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
		panic("qr.EncodeData: bitstream not byte-aligned")
	}
	if len(bw.buf) != n+3 {
		panic("qr.EncodeData: unexpected forced-section length")
	}

	codewords = make([]sym.Byte, DataCodewordsV11M)
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

// bitWriter packs bits MSB-first into a growing byte slice.
type bitWriter struct {
	buf    []byte
	bitPos int // next bit position within the current byte, 0..7
}

func newBitWriter(byteCap int) *bitWriter {
	return &bitWriter{buf: make([]byte, 0, byteCap)}
}

// write appends the low n bits of value, MSB-first.
func (w *bitWriter) write(value uint, n int) {
	for i := n - 1; i >= 0; i-- {
		if w.bitPos == 0 {
			w.buf = append(w.buf, 0)
		}
		bit := byte((value >> uint(i)) & 1)
		w.buf[len(w.buf)-1] |= bit << uint(7-w.bitPos)
		w.bitPos = (w.bitPos + 1) & 7
	}
}
