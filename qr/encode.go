package qr

import (
	"fmt"

	"github.com/rumo-lunar/qrlogo/qr/spec"
)

// EncodeBytes builds the complete data + EC codeword stream for s
// carrying payload, interleaved per ISO/IEC 18004 §7.6 and padded
// with the version's remainder bits.
//
// The returned slice has length s.TotalCodewords(); the high
// remainder bits (0..7) for versions that need them are appended as
// trailing zero bits inside the final byte. Callers that need the
// exact bit stream for placement use [BitStream].
//
// Returns an error if payload exceeds s.MaxByteModePayload().
func EncodeBytes(payload []byte, s spec.Spec) ([]byte, error) {
	if max := s.MaxByteModePayload(); len(payload) > max {
		return nil, fmt.Errorf(
			"qr: payload of %d bytes exceeds %s budget of %d",
			len(payload), s, max)
	}
	data := frameAndPad(payload, s)
	return interleave(data, s), nil
}

// BitStream returns EncodeBytes(payload, s) as a bit slice of length
//
//	s.TotalCodewords()*8 + s.Version.RemainderBits()
//
// suitable for the zig-zag placement pass. Each entry is 0 or 1.
func BitStream(payload []byte, s spec.Spec) ([]byte, error) {
	cw, err := EncodeBytes(payload, s)
	if err != nil {
		return nil, err
	}
	rem := RemainderBits(s.Version)
	bits := make([]byte, len(cw)*8+rem)
	for i, b := range cw {
		for j := 0; j < 8; j++ {
			bits[i*8+j] = (b >> uint(7-j)) & 1
		}
	}
	// Remainder bits stay zero.
	return bits, nil
}

// frameAndPad builds the unmasked data codeword stream for payload at
// spec s. Length = s.DataCodewords(). Layout:
//
//	mode(4) | charCount(8|16) | payload(8N) | terminator(≤4) | bit-pad |
//	pad bytes (0xEC, 0x11, alternating) to fill DataCodewords
func frameAndPad(payload []byte, s spec.Spec) []byte {
	totalDataBits := s.DataCodewords() * 8

	w := newBitWriter(s.DataCodewords())
	w.write(0b0100, 4) // byte mode indicator
	w.write(uint(len(payload)), s.Version.ByteModeCharCountBits())
	for _, b := range payload {
		w.write(uint(b), 8)
	}

	// Terminator: up to 4 zero bits, but never past totalDataBits.
	term := totalDataBits - w.nbits
	if term > 4 {
		term = 4
	}
	if term > 0 {
		w.write(0, term)
	}

	// Bit-pad to next byte boundary.
	if r := w.nbits % 8; r != 0 {
		w.write(0, 8-r)
	}

	// Byte-pad with alternating 0xEC, 0x11.
	pad := []byte{0xEC, 0x11}
	for i := 0; len(w.bytes) < s.DataCodewords(); i++ {
		w.bytes = append(w.bytes, pad[i%2])
	}
	return w.bytes
}

// interleave splits data into RS blocks per s, computes the EC
// codewords for each block, and produces the column-major interleaved
// stream prescribed by ISO/IEC 18004 §7.6.
//
//	output = [data[0] of every block, then data[1] of every block, …]
//	         ++ [ec[0] of every block, then ec[1] of every block, …]
//
// When G2 blocks are larger than G1 blocks, the short G1 blocks have
// no contribution to the trailing data columns.
func interleave(data []byte, s spec.Spec) []byte {
	g1Blocks, g1Size, g2Blocks, g2Size := s.Blocks()
	ecPerBlock := s.ECPerBlock()
	totalBlocks := g1Blocks + g2Blocks

	// Slice data into per-block buffers and compute EC per block.
	dataBlocks := make([][]byte, 0, totalBlocks)
	ecBlocks := make([][]byte, 0, totalBlocks)
	offset := 0
	for i := 0; i < g1Blocks; i++ {
		b := data[offset : offset+g1Size]
		offset += g1Size
		dataBlocks = append(dataBlocks, b)
		ecBlocks = append(ecBlocks, EncodeRS(b, ecPerBlock))
	}
	for i := 0; i < g2Blocks; i++ {
		b := data[offset : offset+g2Size]
		offset += g2Size
		dataBlocks = append(dataBlocks, b)
		ecBlocks = append(ecBlocks, EncodeRS(b, ecPerBlock))
	}

	maxData := g1Size
	if g2Size > maxData {
		maxData = g2Size
	}

	out := make([]byte, 0, s.TotalCodewords())

	// Interleave data column-major: data[col] of each block in order.
	for col := 0; col < maxData; col++ {
		for _, b := range dataBlocks {
			if col < len(b) {
				out = append(out, b[col])
			}
		}
	}
	// Interleave EC column-major: all blocks always have ecPerBlock EC.
	for col := 0; col < ecPerBlock; col++ {
		for _, b := range ecBlocks {
			out = append(out, b[col])
		}
	}
	return out
}

// RemainderBits returns the number of trailing zero bits appended to
// the codeword stream for version v, per ISO/IEC 18004 Table 1.
func RemainderBits(v spec.Version) int {
	switch {
	case v == 1:
		return 0
	case v >= 2 && v <= 6:
		return 7
	case v >= 7 && v <= 13:
		return 0
	case v >= 14 && v <= 20:
		return 3
	case v >= 21 && v <= 27:
		return 4
	case v >= 28 && v <= 34:
		return 3
	case v >= 35 && v <= 40:
		return 0
	}
	panic(fmt.Sprintf("qr.RemainderBits: invalid version %d", v))
}

// bitWriter packs bits MSB-first into a growing byte slice.
type bitWriter struct {
	bytes []byte
	nbits int
}

func newBitWriter(cap int) *bitWriter {
	return &bitWriter{bytes: make([]byte, 0, cap)}
}

// write appends the low n bits of value, MSB-first.
func (w *bitWriter) write(value uint, n int) {
	for i := n - 1; i >= 0; i-- {
		if w.nbits%8 == 0 {
			w.bytes = append(w.bytes, 0)
		}
		bit := byte((value >> uint(i)) & 1)
		w.bytes[w.nbits/8] |= bit << uint(7-w.nbits%8)
		w.nbits++
	}
}
