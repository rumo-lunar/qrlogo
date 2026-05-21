package spec

import "testing"

func TestSpec_V40M_MatchesLegacyConstants(t *testing.T) {
	// Arrange
	sut := Spec{Version: 40, EC: ECMedium}

	// Assert: legacy hardcoded values from the V40-M-only encoder.
	if got := sut.Version.Size(); got != 177 {
		t.Errorf("Size = %d, want 177", got)
	}
	if got := sut.DataCodewords(); got != 2334 {
		t.Errorf("DataCodewords = %d, want 2334", got)
	}
	if got := sut.ECCodewords(); got != 1372 {
		t.Errorf("ECCodewords = %d, want 1372", got)
	}
	if got := sut.TotalCodewords(); got != 3706 {
		t.Errorf("TotalCodewords = %d, want 3706", got)
	}
	if got := sut.ECPerBlock(); got != 28 {
		t.Errorf("ECPerBlock = %d, want 28", got)
	}
	if got := sut.BlockCount(); got != 49 {
		t.Errorf("BlockCount = %d, want 49", got)
	}
	if got := sut.MaxByteModePayload(); got != 2331 {
		t.Errorf("MaxByteModePayload = %d, want 2331", got)
	}
	if got := sut.FormatInfo(2); got != 0b101111001111100 {
		t.Errorf("FormatInfo(2) = 0b%015b, want 0b101111001111100", got)
	}
	if got := Version(40).VersionInfo(); got != 0b101000110001101001 {
		t.Errorf("VersionInfo(40) = 0b%018b, want 0b101000110001101001", got)
	}
}

func TestSpec_FormatInfo_KnownValues(t *testing.T) {
	// Reference values from ISO/IEC 18004 Annex C, Table C.1.
	// L=01, M=00, Q=11, H=10 (Table 12); mask is the 3 LSBs.
	cases := []struct {
		ec   ECLevel
		mask int
		want uint16
	}{
		{ECLow, 0, 0x77C4},
		{ECLow, 7, 0x6976},
		{ECMedium, 0, 0x5412},
		{ECMedium, 2, 0b101111001111100},
		{ECQuartile, 0, 0x355F},
		{ECHigh, 0, 0x1689},
		{ECHigh, 7, 0x083B},
	}
	for _, c := range cases {
		s := Spec{Version: 1, EC: c.ec}
		if got := s.FormatInfo(c.mask); got != c.want {
			t.Errorf("FormatInfo(EC=%s, mask=%d) = 0x%04x, want 0x%04x",
				c.ec, c.mask, got, c.want)
		}
	}
}

func TestSpec_VersionInfo_NoneBelow7(t *testing.T) {
	for v := MinVersion; v <= 6; v++ {
		if got := v.VersionInfo(); got != 0 {
			t.Errorf("V%d VersionInfo = %#x, want 0", v, got)
		}
		if v.HasVersionInfo() {
			t.Errorf("V%d HasVersionInfo() = true, want false", v)
		}
	}
	for v := Version(7); v <= MaxVersion; v++ {
		if !v.HasVersionInfo() {
			t.Errorf("V%d HasVersionInfo() = false, want true", v)
		}
		if got := v.VersionInfo(); got == 0 {
			t.Errorf("V%d VersionInfo = 0, want non-zero", v)
		}
	}
}

func TestSpec_VersionInfo_KnownValues(t *testing.T) {
	// ISO/IEC 18004 Annex D, Table D.1 (selection).
	cases := []struct {
		v    Version
		want uint32
	}{
		{7, 0x07C94},
		{8, 0x085BC},
		{9, 0x09A99},
		{10, 0x0A4D3},
		{14, 0x0E60D},
		{20, 0x149A6},
		{40, 0x28C69},
	}
	for _, c := range cases {
		if got := c.v.VersionInfo(); got != c.want {
			t.Errorf("V%d VersionInfo = %#x, want %#x", c.v, got, c.want)
		}
	}
}

func TestSpec_MaxByteModePayload_AllVersions(t *testing.T) {
	// Spot-checks against the byte-mode capacity table in
	// ISO/IEC 18004 Annex D, Table 7.
	cases := []struct {
		v    Version
		ec   ECLevel
		want int
	}{
		{1, ECLow, 17}, {1, ECMedium, 14}, {1, ECQuartile, 11}, {1, ECHigh, 7},
		{5, ECHigh, 44},
		{10, ECHigh, 119},
		{20, ECHigh, 382},
		{40, ECLow, 2953}, {40, ECMedium, 2331}, {40, ECQuartile, 1663}, {40, ECHigh, 1273},
	}
	for _, c := range cases {
		s := Spec{Version: c.v, EC: c.ec}
		if got := s.MaxByteModePayload(); got != c.want {
			t.Errorf("V%d%s MaxByteModePayload = %d, want %d", c.v, c.ec, got, c.want)
		}
	}
}

func TestSpec_AutoFit_PicksSmallestVersion(t *testing.T) {
	// 100-byte payload at EC H:
	// V9-H cap = 98 (too small), V10-H cap = 119 (fits) → V10.
	v, err := AutoFit(100, ECHigh)
	if err != nil {
		t.Fatalf("AutoFit(100, H) error: %v", err)
	}
	if v != 10 {
		t.Errorf("AutoFit(100, H) = V%d, want V10", v)
	}

	// 50 bytes at EC H: V6-H cap = 60 fits.
	v, err = AutoFit(50, ECHigh)
	if err != nil {
		t.Fatalf("AutoFit(50, H) error: %v", err)
	}
	if v != 6 {
		t.Errorf("AutoFit(50, H) = V%d, want V6", v)
	}

	// 1 byte at EC L fits in V1.
	v, err = AutoFit(1, ECLow)
	if err != nil {
		t.Fatalf("AutoFit(1, L) error: %v", err)
	}
	if v != 1 {
		t.Errorf("AutoFit(1, L) = V%d, want V1", v)
	}
}

func TestSpec_AutoFit_OverflowErrors(t *testing.T) {
	if _, err := AutoFit(5000, ECHigh); err == nil {
		t.Error("AutoFit(5000, H) returned nil error, want overflow")
	}
}
