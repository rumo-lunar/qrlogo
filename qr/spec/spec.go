package spec

import "fmt"

// Spec is the (Version, EC level) pair that fully determines the
// capacity, block layout and format-info bits of a QR symbol.
//
// Mask is not part of Spec because the mask choice is independent
// of the data layout — the same Spec produces 8 different rendered
// symbols, one per mask.
type Spec struct {
	Version Version
	EC      ECLevel
}

// New constructs a validated Spec.
func New(v Version, ec ECLevel) (Spec, error) {
	if _, err := NewVersion(int(v)); err != nil {
		return Spec{}, err
	}
	return Spec{Version: v, EC: ec}, nil
}

// String renders Spec as "VxxC" (e.g. "V10H").
func (s Spec) String() string {
	return fmt.Sprintf("V%d%s", s.Version, s.EC)
}

// layout returns the RS block layout for this Spec.
func (s Spec) layout() blockLayout {
	return blockTable[int(s.Version)-1][s.EC]
}

// ECPerBlock is the number of error-correction codewords per block.
func (s Spec) ECPerBlock() int { return s.layout().ECPerBlock }

// Blocks reports the RS block grouping. The returned values satisfy:
//
//	DataCodewords() == g1Blocks*g1Size + g2Blocks*g2Size
//
// and every block carries the same ECPerBlock() EC codewords.
//
// For specs where every block has the same size, g2Blocks == 0 and
// g2Size == 0.
func (s Spec) Blocks() (g1Blocks, g1Size, g2Blocks, g2Size int) {
	l := s.layout()
	return l.G1Blocks, l.G1Codewords, l.G2Blocks, l.G2Codewords
}

// BlockCount returns the total number of RS blocks.
func (s Spec) BlockCount() int {
	l := s.layout()
	return l.G1Blocks + l.G2Blocks
}

// DataCodewords is the total number of data codewords carried by
// the symbol (across all RS blocks).
func (s Spec) DataCodewords() int {
	l := s.layout()
	return l.G1Blocks*l.G1Codewords + l.G2Blocks*l.G2Codewords
}

// ECCodewords is the total number of EC codewords across all blocks.
func (s Spec) ECCodewords() int {
	l := s.layout()
	return (l.G1Blocks + l.G2Blocks) * l.ECPerBlock
}

// TotalCodewords is DataCodewords + ECCodewords (also the number of
// codewords laid out in the matrix; matches Annex D Table 7 totals).
func (s Spec) TotalCodewords() int {
	return s.DataCodewords() + s.ECCodewords()
}

// MaxByteModePayload returns the maximum byte-mode payload length
// (in bytes) that fits in this Spec, accounting for the 4-bit mode
// indicator, the version-dependent character-count indicator and
// the 4-bit terminator (which we always emit).
//
//	free bits = DataCodewords*8 − 4 − charCountBits − 4
//	max bytes = free bits / 8
func (s Spec) MaxByteModePayload() int {
	overheadBits := 4 + s.Version.ByteModeCharCountBits() + 4
	return (s.DataCodewords()*8 - overheadBits) / 8
}

// FormatInfo returns the 15-bit format-information string for this
// Spec under mask m (m ∈ [0, 7]). See ISO/IEC 18004 §7.9:
//
//  1. 5-bit data = (ecBits << 3) | mask
//  2. BCH(15, 5) by G(x) = x¹⁰+x⁸+x⁵+x⁴+x²+x+1 over GF(2)
//  3. XOR final 15 bits with 0x5412 to avoid the all-zero string.
//
// MSB-first; bit 14 is the leftmost bit when written into the grid.
func (s Spec) FormatInfo(mask int) uint16 {
	if mask < 0 || mask > 7 {
		panic("spec: mask out of range [0, 7]")
	}
	data5 := uint32((ecLevelBits(s.EC) << 3) | uint(mask))
	bits := data5 << 10
	rem := bits
	const g uint32 = 0b10100110111 // x^10 + x^8 + x^5 + x^4 + x^2 + x + 1
	for i := 14; i >= 10; i-- {
		if rem&(1<<uint(i)) != 0 {
			rem ^= g << uint(i-10)
		}
	}
	return uint16((bits | rem) ^ 0x5412)
}

// AutoFit returns the smallest Version at the given EC level that
// accommodates a byte-mode payload of payloadBytes bytes. Returns
// an error if no version up to V40 is large enough.
func AutoFit(payloadBytes int, ec ECLevel) (Version, error) {
	if payloadBytes < 0 {
		return 0, fmt.Errorf("spec: negative payload length %d", payloadBytes)
	}
	for v := MinVersion; v <= MaxVersion; v++ {
		s := Spec{Version: v, EC: ec}
		if s.MaxByteModePayload() >= payloadBytes {
			return v, nil
		}
	}
	return 0, fmt.Errorf(
		"spec: payload of %d bytes does not fit any version at EC %s",
		payloadBytes, ec)
}
