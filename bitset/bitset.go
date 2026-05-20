// Package bitset provides linear algebra over GF(2) for the qrlogo
// constraint solver.
//
// All equations have the form
//
//	x_{i₁} ⊕ x_{i₂} ⊕ … ⊕ x_{iₖ} = Target
//
// where each xᵢ ∈ {0, 1} and ⊕ is XOR. Variables are packed into
// []uint64 bitsets: variable x_n occupies bit (n mod 64) of
// Vars[n/64], where bit 0 is the least significant bit of the word.
//
// The solver implements Gauss–Jordan elimination over GF(2), exploiting
// the fact that all row operations collapse to a single bitwise XOR of
// two []uint64 slices plus a XOR of their Target bytes.
package bitset

import mathbits "math/bits"

// Row is one linear equation over GF(2).
//
// Vars must have length equal to (System.NumVars + 63) / 64. The
// packing convention is LSB-first within each word: variable x_n is
//
//	(Vars[n/64] >> uint(n%64)) & 1
//
// Target is either 0 or 1.
type Row struct {
	Vars   []uint64
	Target byte
}

// System is a collection of linear equations over the same fixed set
// of variables. Rows may appear in any order; Solve treats them as a
// set.
type System struct {
	NumVars int
	Rows    []Row

	// Seed, when non-nil, is used to assign free variables (those with
	// no pivot after elimination) instead of defaulting them to 0.
	// Bit n of Seed (MSB-first per byte) is assigned to free variable n.
	// Pivot variables are adjusted accordingly so all constraints are
	// still satisfied. Must be at least ceil(NumVars/8) bytes; excess
	// bytes are ignored.
	Seed []byte
}

// workRow is the mutable per-row state used during Gauss–Jordan.
type workRow struct {
	vars    []uint64
	target  byte
	origIdx int
}

// buildWork copies s.Rows into a fresh mutable slice.
func buildWork(s *System, words int) []workRow {
	work := make([]workRow, len(s.Rows))
	for i, r := range s.Rows {
		v := make([]uint64, words)
		copy(v, r.Vars)
		work[i] = workRow{vars: v, target: r.Target & 1, origIdx: i}
	}
	return work
}

// gauss performs full Gauss–Jordan elimination over GF(2) in place.
// Returns the pivot column indices (length = rank) and the rank r;
// work[0..r-1] are the pivot rows in column order after the call.
func gauss(work []workRow, numVars, words int) (pivotCol []int, rank int) {
	pivotCol = make([]int, 0, numVars)
	m := len(work)
	r := 0
	for c := 0; c < numVars; c++ {
		wordIdx := c / 64
		bitMask := uint64(1) << uint(c%64)

		pivot := -1
		for i := r; i < m; i++ {
			if work[i].vars[wordIdx]&bitMask != 0 {
				pivot = i
				break
			}
		}
		if pivot == -1 {
			continue
		}
		if pivot != r {
			work[r], work[pivot] = work[pivot], work[r]
		}
		for j := 0; j < m; j++ {
			if j == r || work[j].vars[wordIdx]&bitMask == 0 {
				continue
			}
			for k := 0; k < words; k++ {
				work[j].vars[k] ^= work[r].vars[k]
			}
			work[j].target ^= work[r].target
		}
		pivotCol = append(pivotCol, c)
		r++
	}
	return pivotCol, r
}

// extractSolution reads the solution from the reduced pivot rows.
//
// Free variables (columns without a pivot) are assigned from seed
// (MSB-first per byte: bit n = (seed[n/8] >> (7-n%8)) & 1). When
// seed is nil they default to 0, preserving the original behaviour.
//
// Pivot variables are computed from their reduced rows, with each
// free variable's seed contribution XOR'd in so that all constraints
// remain satisfied.
//
// pivotRows must be work[0..rank-1] after gauss() — one entry per
// pivot column, in the same order as pivotCol.
func extractSolution(numVars int, pivotRows []workRow, pivotCol []int, seed []byte) []byte {
	bits := make([]byte, (numVars+7)/8)

	isPivot := make([]bool, numVars)
	for _, c := range pivotCol {
		isPivot[c] = true
	}

	seedBit := func(col int) byte {
		if col/8 >= len(seed) {
			return 0
		}
		return (seed[col/8] >> uint(7-col%8)) & 1
	}

	// Assign free variables from seed.
	for col := 0; col < numVars; col++ {
		if !isPivot[col] && seedBit(col) != 0 {
			bits[col/8] |= 1 << uint(7-col%8)
		}
	}

	// Compute each pivot variable, accounting for free variable noise.
	// After full Gauss–Jordan, work[k].vars has a 1 only in pivotCol[k]
	// and in free variable columns. Summing the seed bits of all free
	// variable columns present in row k gives the correction.
	for k, col := range pivotCol {
		val := pivotRows[k].target
		for w, v := range pivotRows[k].vars {
			for v != 0 {
				bit := uint(mathbits.TrailingZeros64(v))
				globalCol := w*64 + int(bit)
				if globalCol < numVars && !isPivot[globalCol] {
					val ^= seedBit(globalCol)
				}
				v &^= 1 << bit
			}
		}
		if val != 0 {
			bits[col/8] |= 1 << uint(7-col%8)
		}
	}

	return bits
}

// Solve performs Gauss–Jordan elimination over GF(2) and returns:
//
//   - bits: the packed solution, of length ceil(NumVars/8). Bits are
//     stored MSB-first within each byte: variable x_n is
//
//     (bits[n/8] >> uint(7-n%8)) & 1
//
//     to match the bit ordering used by QR data placement.
//
//   - conflictRow: when ok is false, the index (in the input s.Rows)
//     of a row that was reduced to the contradiction 0 = 1. When ok
//     is true this value is meaningless.
//
//   - ok: true if the system is consistent. Underdetermined systems
//     are consistent: free variables are seeded from s.Seed (or 0).
//
// Solve does not mutate s; it operates on an internal copy.
func (s *System) Solve() (bits []byte, conflictRow int, ok bool) {
	words := (s.NumVars + 63) / 64
	work := buildWork(s, words)
	pivotCol, r := gauss(work, s.NumVars, words)
	m := len(work)

	for i := r; i < m; i++ {
		if work[i].target == 0 {
			continue
		}
		allZero := true
		for k := 0; k < words; k++ {
			if work[i].vars[k] != 0 {
				allZero = false
				break
			}
		}
		if allZero {
			return nil, work[i].origIdx, false
		}
	}

	return extractSolution(s.NumVars, work[:r], pivotCol, s.Seed), 0, true
}

// SolveBestEffort runs the same Gauss–Jordan elimination as Solve but
// instead of returning false on a contradicting row it silently drops
// that row and continues. Returns the solution bits and the number of
// rows that were dropped.
func (s *System) SolveBestEffort() (bits []byte, dropped int) {
	words := (s.NumVars + 63) / 64
	work := buildWork(s, words)
	pivotCol, r := gauss(work, s.NumVars, words)
	m := len(work)

	for i := r; i < m; i++ {
		if work[i].target == 0 {
			continue
		}
		allZero := true
		for k := 0; k < words; k++ {
			if work[i].vars[k] != 0 {
				allZero = false
				break
			}
		}
		if allZero {
			work[i].target = 0
			dropped++
		}
	}

	return extractSolution(s.NumVars, work[:r], pivotCol, s.Seed), dropped
}
