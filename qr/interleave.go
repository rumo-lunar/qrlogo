package qr

import (
	"github.com/rumo-lunar/qrlogo/qr/sym"
)

// V40-M block structure (per ISO/IEC 18004 Annex D, Table 9):
//
//	Group 1: 18 blocks × 47 data codewords
//	Group 2: 31 blocks × 48 data codewords
//	EC:      49 blocks × 28 EC codewords each
//	Total:   2334 data + 1372 EC = 3706 codewords
const (
	blocks      = 49
	ecPerBlock  = 28
	shortBlock  = 47 // group 1 data codewords per block
	longBlock   = 48 // group 2 data codewords per block
	shortBlocks = 18 // number of group-1 blocks
)

// blockBounds returns the [start, end) data-slice indices for
// the j-th block (0 ≤ j < 49). Blocks 0..17 are the group-1 blocks
// of 47 codewords each; blocks 18..48 each hold 48 codewords.
func blockBounds(j int) (start, end int) {
	if j < shortBlocks {
		start = j * shortBlock
		end = start + shortBlock
		return
	}
	start = shortBlocks*shortBlock + (j-shortBlocks)*longBlock
	end = start + longBlock
	return
}

// Interleave takes a complete V40-M data-codeword stream of
// length DataCodewords (2334) and returns the fully interleaved
// data + EC stream of length DataCodewords + ECCodewords
// (3706) ready for placement in the QR matrix.
//
// Per ISO/IEC 18004 §7.6, codewords from each of the 49 RS blocks
// are interleaved column-major:
//
//   - Data: codeword i of block 0, then codeword i of block 1, …,
//     then codeword i of block 48, for i = 0, 1, … until each block
//     is exhausted. When the group-1 blocks (47 codewords) are done
//     at i = 47, they are silently skipped while the group-2 blocks
//     (48 codewords) contribute their last codeword.
//   - EC: same column-major sweep over 28 EC codewords × 49 blocks.
//     All EC blocks have the same length, so no skipping is needed.
//
// EC codewords for each data block are computed via EncodeRS with
// numEC = 28.
//
// Panics if len(data) != DataCodewords.
func Interleave(d *sym.Domain, data []sym.Byte) []sym.Byte {
	if len(data) != DataCodewords {
		panic("qr.Interleave: expected 2334 data codewords")
	}

	// Split into 49 RS blocks and compute EC per block.
	dataBlocks := make([][]sym.Byte, blocks)
	ecs := make([][]sym.Byte, blocks)
	for j := 0; j < blocks; j++ {
		start, end := blockBounds(j)
		dataBlocks[j] = data[start:end]
		ecs[j] = EncodeRS(d, dataBlocks[j], ecPerBlock)
	}

	out := make([]sym.Byte, 0, DataCodewords+ECCodewords)

	// Interleave data column-major. The longest block has longBlock
	// codewords; the short blocks are skipped once i ≥ their length.
	for i := 0; i < longBlock; i++ {
		for j := 0; j < blocks; j++ {
			if i < len(dataBlocks[j]) {
				out = append(out, dataBlocks[j][i])
			}
		}
	}

	// Interleave EC column-major. All blocks have ecPerBlock EC
	// codewords, so no length guard is needed.
	for i := 0; i < ecPerBlock; i++ {
		for j := 0; j < blocks; j++ {
			out = append(out, ecs[j][i])
		}
	}

	return out
}
