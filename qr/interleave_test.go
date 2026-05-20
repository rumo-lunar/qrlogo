package qr_test

import (
	"math/rand"
	"strings"
	"testing"

	"github.com/rumo-lunar/qrlogo/qr"
	"github.com/rumo-lunar/qrlogo/qr/sym"
)

// concreteByteRange is a tiny helper that builds a byte slice
// filled with src[start:end], the concrete-byte mirror of slicing
// a symbolic data array.
func concreteByteRange(src []byte, start, end int) []byte {
	out := make([]byte, end-start)
	copy(out, src[start:end])
	return out
}

func TestInterleaveV11M_OutputLength(t *testing.T) {
	// Arrange
	d := sym.NewDomain(0)
	data := make([]sym.Byte, qr.DataCodewordsV11M)
	for i := range data {
		data[i] = d.ConstByte(byte(i))
	}

	// Act
	out := qr.InterleaveV11M(d, data)

	// Assert
	if want := qr.DataCodewordsV11M + qr.ECCodewordsV11M; len(out) != want {
		t.Fatalf("len(out) = %d, want %d", len(out), want)
	}
}

func TestInterleaveV11M_DataOrderSpotChecks(t *testing.T) {
	// Arrange: codeword i carries byte value i mod 256.
	d := sym.NewDomain(0)
	data := make([]sym.Byte, qr.DataCodewordsV11M)
	for i := range data {
		data[i] = d.ConstByte(byte(i))
	}

	// Act
	out := qr.InterleaveV11M(d, data)
	sol := []byte{}

	// Assert: block bounds are [0,50), [50,101), [101,152), [152,203), [203,254).
	// At column i, blocks emit data[i], data[50+i], data[101+i], data[152+i],
	// data[203+i] in that order. When i = 50, block 0 is skipped.
	cases := []struct {
		pos  int
		want byte
	}{
		{0, byte(0)},       // block 0 col 0
		{1, byte(50)},      // block 1 col 0
		{2, byte(101)},     // block 2 col 0
		{3, byte(152)},     // block 3 col 0
		{4, byte(203)},     // block 4 col 0
		{5, byte(1)},       // block 0 col 1
		{6, byte(51)},      // block 1 col 1
		{249, byte(252)},   // block 4 col 49: data[203+49] = data[252]
		{250, byte(100)},   // block 1 col 50: data[50+50] = data[100]; block 0 skipped
		{251, byte(151)},   // block 2 col 50
		{252, byte(202)},   // block 3 col 50
		{253, byte(253)},   // block 4 col 50: last data codeword
	}
	for _, c := range cases {
		if got := d.ResolveByte(out[c.pos], sol); got != c.want {
			t.Errorf("out[%d] = 0x%02x, want 0x%02x", c.pos, got, c.want)
		}
	}
}

func TestInterleaveV11M_ECMatchesPerBlockReference(t *testing.T) {
	// Arrange: distinct constant data so the EC of each block is
	// deterministic and distinct from its neighbours.
	d := sym.NewDomain(0)
	rawData := make([]byte, qr.DataCodewordsV11M)
	rng := rand.New(rand.NewSource(20260520))
	for i := range rawData {
		rawData[i] = byte(rng.Intn(256))
	}
	data := make([]sym.Byte, qr.DataCodewordsV11M)
	for i, b := range rawData {
		data[i] = d.ConstByte(b)
	}

	// Compute reference EC for each of the 5 blocks via the concrete
	// referenceRS already exercised by rs_test.go.
	blockStarts := []int{0, 50, 101, 152, 203}
	blockEnds := []int{50, 101, 152, 203, 254}
	refEC := make([][]byte, 5)
	for j := 0; j < 5; j++ {
		refEC[j] = referenceRS(concreteByteRange(rawData, blockStarts[j], blockEnds[j]), 30)
	}

	// Act
	out := qr.InterleaveV11M(d, data)
	sol := []byte{}

	// Assert: EC region starts at position 254. EC codewords are
	// interleaved column-major: out[254 + i*5 + j] == refEC[j][i].
	const ecStart = 254
	for i := 0; i < 30; i++ {
		for j := 0; j < 5; j++ {
			pos := ecStart + i*5 + j
			got := d.ResolveByte(out[pos], sol)
			want := refEC[j][i]
			if got != want {
				t.Errorf("EC i=%d j=%d (out[%d]) = 0x%02x, want 0x%02x",
					i, j, pos, got, want)
			}
		}
	}
}

func TestInterleaveV11M_FullPipelineLinearity(t *testing.T) {
	// Arrange: real URL through the full pipeline EncodeData ->
	// InterleaveV11M, then resolve with several random solutions and
	// check that each resolved output matches a concrete-byte
	// reconstruction (concrete data + concrete RS + interleave).
	url := strings.Repeat("a", 50) // 50 chars → 53 forced codewords + 201 padding
	data, d := qr.EncodeData(url)
	symOut := qr.InterleaveV11M(d, data)

	const (
		dataLen = 254
		ecLen   = 150
		outLen  = dataLen + ecLen
	)

	if len(symOut) != outLen {
		t.Fatalf("len(symOut) = %d, want %d", len(symOut), outLen)
	}

	rng := rand.New(rand.NewSource(1))
	for trial := 0; trial < 3; trial++ {
		// Random solution covering all free variables.
		sol := make([]byte, (d.NumVars+7)/8)
		for i := range sol {
			sol[i] = byte(rng.Intn(256))
		}

		// Step 1: resolve the 254 data codewords back to concrete bytes.
		resolvedData := make([]byte, dataLen)
		for i := 0; i < dataLen; i++ {
			resolvedData[i] = d.ResolveByte(data[i], sol)
		}

		// Step 2: concrete reference — split into blocks, compute EC,
		// interleave column-major.
		blockStarts := []int{0, 50, 101, 152, 203}
		blockEnds := []int{50, 101, 152, 203, 254}
		blockBytes := make([][]byte, 5)
		ecBytes := make([][]byte, 5)
		for j := 0; j < 5; j++ {
			blockBytes[j] = concreteByteRange(resolvedData, blockStarts[j], blockEnds[j])
			ecBytes[j] = referenceRS(blockBytes[j], 30)
		}

		ref := make([]byte, 0, outLen)
		for i := 0; i < 51; i++ {
			for j := 0; j < 5; j++ {
				if i < len(blockBytes[j]) {
					ref = append(ref, blockBytes[j][i])
				}
			}
		}
		for i := 0; i < 30; i++ {
			for j := 0; j < 5; j++ {
				ref = append(ref, ecBytes[j][i])
			}
		}

		// Step 3: resolve the symbolic interleaved output and compare.
		for i := 0; i < outLen; i++ {
			got := d.ResolveByte(symOut[i], sol)
			if got != ref[i] {
				t.Fatalf("trial %d out[%d] = 0x%02x, want 0x%02x",
					trial, i, got, ref[i])
			}
		}
	}
}

func TestInterleaveV11M_PanicsOnWrongLength(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("expected panic on wrong-length data")
		}
	}()
	d := sym.NewDomain(0)
	qr.InterleaveV11M(d, make([]sym.Byte, 100))
}
