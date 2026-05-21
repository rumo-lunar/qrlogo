package qr

// layout.go centralises the geometric constants of a V40 QR symbol
// so they are defined exactly once and consumed by both the Kind map
// builder (module.go) and the concrete bit-pattern builder
// (function.go).

// finderOrigins is the top-left coordinate of each 7×7 finder pattern.
// V40 has finders at three corners; the bottom-right slot is occupied
// by data and alignment patterns instead.
var finderOrigins = [3][2]int{
	{0, 0},
	{0, Size - 7},
	{Size - 7, 0},
}

// alignmentCentres holds the row/column coordinates of every V40
// alignment-pattern centre (ISO/IEC 18004 Annex E).
var alignmentCentres = [...]int{6, 30, 58, 86, 114, 142, 170}

// alignmentExcluded reports whether the (ar, ac) centre coincides
// with a finder corner and must be skipped when laying out alignment
// patterns.
func alignmentExcluded(ar, ac int) bool {
	first := alignmentCentres[0]
	last := alignmentCentres[len(alignmentCentres)-1]
	return (ar == first && ac == first) ||
		(ar == first && ac == last) ||
		(ar == last && ac == first)
}

// forEachAlignment iterates the 46 alignment-pattern centres that
// actually appear in a V40 symbol (the full 7×7 grid minus the three
// finder-corner exclusions), invoking fn for each.
func forEachAlignment(fn func(ar, ac int)) {
	for _, ar := range alignmentCentres {
		for _, ac := range alignmentCentres {
			if alignmentExcluded(ar, ac) {
				continue
			}
			fn(ar, ac)
		}
	}
}
