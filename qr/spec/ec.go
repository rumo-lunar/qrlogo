// Package spec encodes the per-version, per-EC-level constants and
// tables of ISO/IEC 18004 (QR Code 2005) for versions 1 through 40.
//
// Every other package consumes Spec for its dimensions, codeword
// budgets, block layout, format information and mask selection.
package spec

import "fmt"

// ECLevel is the error-correction level of a QR symbol.
// Recovery capacities are nominal:
//
//	L ≈  7%   M ≈ 15%   Q ≈ 25%   H ≈ 30%
//
// For symbols carrying a centre logo, H is the only level that
// reliably tolerates the logo's obscuration plus camera noise.
type ECLevel int

const (
	ECLow      ECLevel = iota // L
	ECMedium                  // M
	ECQuartile                // Q
	ECHigh                    // H
)

// String returns the single-letter spec name.
func (e ECLevel) String() string {
	switch e {
	case ECLow:
		return "L"
	case ECMedium:
		return "M"
	case ECQuartile:
		return "Q"
	case ECHigh:
		return "H"
	}
	return "?"
}

// ParseECLevel parses "L", "M", "Q", "H" (case-insensitive).
func ParseECLevel(s string) (ECLevel, error) {
	switch s {
	case "L", "l":
		return ECLow, nil
	case "M", "m":
		return ECMedium, nil
	case "Q", "q":
		return ECQuartile, nil
	case "H", "h":
		return ECHigh, nil
	}
	return 0, fmt.Errorf("spec: unknown EC level %q (want L, M, Q or H)", s)
}

// ecLevelBits returns the 2-bit field used in the format-info string
// for level e (ISO/IEC 18004 Table 12).
func ecLevelBits(e ECLevel) uint {
	switch e {
	case ECLow:
		return 0b01
	case ECMedium:
		return 0b00
	case ECQuartile:
		return 0b11
	case ECHigh:
		return 0b10
	}
	panic("spec: invalid ECLevel")
}
