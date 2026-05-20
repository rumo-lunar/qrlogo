package qr

import (
	"github.com/rumo-lunar/qrlogo/qr/sym"
)

// V11-M block structure (per ISO/IEC 18004 Annex D, Table 9):
//
//	Group 1: 1 block × 50 data codewords
//	Group 2: 4 blocks × 51 data codewords
//	EC:      5 blocks × 30 EC codewords each
//	Total:   254 data + 150 EC = 404 codewords
//
// These constants exist alongside DataCodewordsV11M / ECCodewordsV11M
// so the interleaver doesn't carry magic numbers in its loops.
const (
	v11mBlocks      = 5
	v11mECPerBlock  = 30
	v11mShortBlock  = 50 // group 1 block size
	v11mLongBlock   = 51 // group 2 block size
	v11mShortBlocks = 1  // number of group-1 blocks
)

// blockBoundsV11M returns the [start, end) data-slice indices for
// the j-th block (0 ≤ j < 5). Block 0 is the lone group-1 block of
// 50 codewords; blocks 1..4 each hold 51 codewords.
func blockBoundsV11M(j int) (start, end int) {
	if j == 0 {
		return 0, v11mShortBlock
	}
	start = v11mShortBlock + (j-1)*v11mLongBlock
	end = start + v11mLongBlock
	return
}

// InterleaveV11M takes a complete V11-M data-codeword stream of
// length DataCodewordsV11M (254) and returns the fully interleaved
// data + EC stream of length DataCodewordsV11M + ECCodewordsV11M
// (404) ready for placement in the QR matrix.
//
// Per ISO/IEC 18004 §7.6, codewords from each of the 5 RS blocks
// are interleaved column-major:
//
//   - Data: codeword i of block 0, then codeword i of block 1, …,
//     then codeword i of block 4, for i = 0, 1, … until each block
//     is exhausted. When block 0 (50 codewords) is done at i = 50,
//     it is silently skipped while the longer blocks (51 codewords)
//     contribute their last codeword.
//   - EC: same column-major sweep over 30 EC codewords × 5 blocks.
//     All EC blocks have the same length, so no skipping is needed.
//
// EC codewords for each data block are computed via EncodeRS with
// numEC = 30.
//
// Panics if len(data) != DataCodewordsV11M.
func InterleaveV11M(d *sym.Domain, data []sym.Byte) []sym.Byte {
	if len(data) != DataCodewordsV11M {
		panic("qr.InterleaveV11M: expected 254 data codewords")
	}

	// Split into 5 RS blocks and compute EC per block.
	blocks := make([][]sym.Byte, v11mBlocks)
	ecs := make([][]sym.Byte, v11mBlocks)
	for j := 0; j < v11mBlocks; j++ {
		start, end := blockBoundsV11M(j)
		blocks[j] = data[start:end]
		ecs[j] = EncodeRS(d, blocks[j], v11mECPerBlock)
	}

	out := make([]sym.Byte, 0, DataCodewordsV11M+ECCodewordsV11M)

	// Interleave data column-major. The longest block has v11mLongBlock
	// codewords; the short block is skipped once i ≥ its length.
	for i := 0; i < v11mLongBlock; i++ {
		for j := 0; j < v11mBlocks; j++ {
			if i < len(blocks[j]) {
				out = append(out, blocks[j][i])
			}
		}
	}

	// Interleave EC column-major. All blocks have v11mECPerBlock EC
	// codewords, so no length guard is needed.
	for i := 0; i < v11mECPerBlock; i++ {
		for j := 0; j < v11mBlocks; j++ {
			out = append(out, ecs[j][i])
		}
	}

	return out
}
