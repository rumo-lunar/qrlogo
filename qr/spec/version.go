package spec

import "fmt"

// Version is a QR symbol version in the closed range [1, 40].
type Version int

// MinVersion and MaxVersion bound the supported version range.
const (
	MinVersion Version = 1
	MaxVersion Version = 40
)

// NewVersion validates v and returns it, or an error.
func NewVersion(v int) (Version, error) {
	if v < int(MinVersion) || v > int(MaxVersion) {
		return 0, fmt.Errorf("spec: version %d out of range [%d, %d]",
			v, MinVersion, MaxVersion)
	}
	return Version(v), nil
}

// Size returns the side length in modules of a Version v symbol:
//
//	n = 4V + 17
//
// For V40 that is 177; for V1, 21.
func (v Version) Size() int {
	return 4*int(v) + 17
}

// DarkModule returns the (row, col) of the single dark module that
// is always 1 (ISO/IEC 18004 §6.10). The cell sits at (4V+9, 8).
func (v Version) DarkModule() (row, col int) {
	return 4*int(v) + 9, 8
}

// AlignmentCentres returns the row/column centres of the alignment
// patterns for version v (ISO/IEC 18004 Annex E). Version 1 returns
// an empty slice.
//
// The returned slice is the cross product of these coordinates with
// itself; callers must skip the three centres that coincide with a
// finder corner using AlignmentExcluded.
func (v Version) AlignmentCentres() []int {
	t := alignmentCentresTable[int(v)-1]
	out := make([]int, len(t))
	copy(out, t)
	return out
}

// AlignmentExcluded reports whether the centre (ar, ac) coincides
// with a finder pattern and must be skipped.
//
// For every version with alignment patterns, the three excluded
// centres are the corners of the alignment grid that overlap the
// top-left, top-right and bottom-left finders.
func (v Version) AlignmentExcluded(ar, ac int) bool {
	centres := alignmentCentresTable[int(v)-1]
	if len(centres) == 0 {
		return false
	}
	first := centres[0]
	last := centres[len(centres)-1]
	return (ar == first && ac == first) ||
		(ar == first && ac == last) ||
		(ar == last && ac == first)
}

// ForEachAlignment iterates the alignment-pattern centres that
// actually appear in version v (the full grid minus the three
// finder-corner exclusions), invoking fn for each.
func (v Version) ForEachAlignment(fn func(ar, ac int)) {
	centres := alignmentCentresTable[int(v)-1]
	for _, ar := range centres {
		for _, ac := range centres {
			if v.AlignmentExcluded(ar, ac) {
				continue
			}
			fn(ar, ac)
		}
	}
}

// FinderOrigins returns the top-left coordinate of each 7×7 finder
// pattern. All versions place finders at the same three corners.
func (v Version) FinderOrigins() [3][2]int {
	n := v.Size()
	return [3][2]int{
		{0, 0},
		{0, n - 7},
		{n - 7, 0},
	}
}

// ByteModeCharCountBits returns the width in bits of the character-
// count indicator used by byte mode for this version (ISO/IEC 18004
// Table 3). Byte mode uses 8 bits for V1–V9 and 16 bits for V10+.
func (v Version) ByteModeCharCountBits() int {
	if v <= 9 {
		return 8
	}
	return 16
}

// HasVersionInfo reports whether the symbol carries the 18-bit
// version-information blocks (only versions ≥ 7).
func (v Version) HasVersionInfo() bool {
	return v >= 7
}

// VersionInfo returns the 18-bit version-information string for v.
// Returns 0 for versions < 7 (no version-info is placed).
//
// The encoding is the BCH(18, 6) code with generator
//
//	g(x) = x¹² + x¹¹ + x¹⁰ + x⁹ + x⁸ + x⁵ + x² + 1
//
// applied to the 6-bit version number, MSB-first.
func (v Version) VersionInfo() uint32 {
	if v < 7 {
		return 0
	}
	const g uint32 = 0b1111100100101 // x^12 + x^11 + x^10 + x^9 + x^8 + x^5 + x^2 + 1
	data := uint32(v) << 12
	rem := data
	for i := 17; i >= 12; i-- {
		if rem&(1<<uint(i)) != 0 {
			rem ^= g << uint(i-12)
		}
	}
	return data | rem
}
