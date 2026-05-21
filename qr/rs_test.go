package qr_test

import (
	"math/rand"
	"testing"

	"github.com/rumo-lunar/qrlogo/qr"
	"github.com/rumo-lunar/qrlogo/qr/gf256"
	"github.com/rumo-lunar/qrlogo/qr/sym"
)

// referenceRS is an independent, concrete-byte Reed–Solomon encoder
// used as the oracle for the symbolic EncodeRS. Algorithm matches the
// classic synthetic polynomial long division.
func referenceRS(data []byte, numEC int) []byte {
	g := gf256.GeneratorPoly(numEC)
	work := make([]byte, len(data)+numEC)
	copy(work, data)
	for i := 0; i < len(data); i++ {
		lead := work[i]
		if lead == 0 {
			continue
		}
		for j := 0; j <= numEC; j++ {
			work[i+j] ^= gf256.Mul(lead, g[j])
		}
	}
	return work[len(data):]
}

func TestEncodeRS_KnownHandTraced(t *testing.T) {
	// Arrange: data = [1, 1, 1, 1], numEC = 2.
	//
	// g_2(x) = x² + 3x + 2.
	// Hand-traced polynomial long division gives EC = [20, 20].
	d := sym.NewDomain(0)
	data := make([]sym.Byte, 4)
	for i := range data {
		data[i] = d.ConstByte(1)
	}

	// Act
	ec := qr.EncodeRS(d, data, 2)

	// Assert
	if len(ec) != 2 {
		t.Fatalf("len(ec) = %d, want 2", len(ec))
	}
	sol := []byte{}
	for i, want := range []byte{20, 20} {
		if got := d.ResolveByte(ec[i], sol); got != want {
			t.Errorf("EC[%d] = %d, want %d", i, got, want)
		}
	}
}

func TestEncodeRS_ConstantData_MatchesReference(t *testing.T) {
	cases := []struct {
		name  string
		data  []byte
		numEC int
	}{
		{"tiny", []byte{0x40, 0x00, 0x16, 0x10}, 7},
		{"deadbeef-10ec", []byte{0xDE, 0xAD, 0xBE, 0xEF}, 10},
		{"v40m-block-size-30ec", make([]byte, 50), 30},
		{"v40m-block-size-51-30ec", makePattern(51, 0xA5), 30},
		{"all-zero-30ec", make([]byte, 51), 30},
		{"all-ones-30ec", makePattern(51, 0xFF), 30},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			// Arrange: no variables; pure constant data.
			d := sym.NewDomain(0)
			symData := make([]sym.Byte, len(c.data))
			for i, b := range c.data {
				symData[i] = d.ConstByte(b)
			}

			// Act
			symEC := qr.EncodeRS(d, symData, c.numEC)

			// Assert
			if len(symEC) != c.numEC {
				t.Fatalf("len(symEC) = %d, want %d", len(symEC), c.numEC)
			}
			sol := []byte{}
			want := referenceRS(c.data, c.numEC)
			for i, w := range want {
				if got := d.ResolveByte(symEC[i], sol); got != w {
					t.Errorf("EC[%d] = 0x%02x, want 0x%02x", i, got, w)
				}
			}
		})
	}
}

func TestEncodeRS_LinearityAgainstReference(t *testing.T) {
	// Arrange: 50 data codewords, the first 24 constant and the
	// next 26 fully symbolic (208 vars). numEC = 30.
	const (
		dataLen  = 50
		constLen = 24
		varLen   = 26
		numEC    = 30
	)
	d := sym.NewDomain(varLen * 8)
	data := make([]sym.Byte, dataLen)
	for i := 0; i < constLen; i++ {
		data[i] = d.ConstByte(byte(i*7 + 1))
	}
	for k := 0; k < varLen; k++ {
		var b sym.Byte
		for j := 0; j < 8; j++ {
			b[j] = d.Variable(k*8 + j)
		}
		data[constLen+k] = b
	}

	// Act once: a single symbolic EC computation that we then resolve
	// against many random solutions.
	symEC := qr.EncodeRS(d, data, numEC)

	// Assert: for each random solution, the resolved EC must equal
	// the concrete RS encoding of the resolved data. If linearity is
	// broken anywhere in EncodeRS, this fails.
	rng := rand.New(rand.NewSource(2026))
	for trial := 0; trial < 8; trial++ {
		sol := make([]byte, varLen)
		for i := range sol {
			sol[i] = byte(rng.Intn(256))
		}

		resolvedData := make([]byte, dataLen)
		for i := range data {
			resolvedData[i] = d.ResolveByte(data[i], sol)
		}
		refEC := referenceRS(resolvedData, numEC)

		for i := 0; i < numEC; i++ {
			got := d.ResolveByte(symEC[i], sol)
			if got != refEC[i] {
				t.Fatalf("trial %d EC[%d]: got 0x%02x, want 0x%02x",
					trial, i, got, refEC[i])
			}
		}
	}
}

func TestEncodeRS_PanicsOnZeroNumEC(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("expected panic on numEC=0")
		}
	}()
	d := sym.NewDomain(0)
	qr.EncodeRS(d, []sym.Byte{d.ConstByte(1)}, 0)
}

func TestEncodeRS_PanicsOnEmptyData(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("expected panic on empty data")
		}
	}()
	d := sym.NewDomain(0)
	qr.EncodeRS(d, nil, 10)
}

func makePattern(n int, b byte) []byte {
	out := make([]byte, n)
	for i := range out {
		out[i] = b
	}
	return out
}
