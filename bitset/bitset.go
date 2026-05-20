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
//     are consistent: free variables default to 0.
//
// Solve does not mutate s; it operates on an internal copy.
func (s *System) Solve() (bits []byte, conflictRow int, ok bool) {
	words := (s.NumVars + 63) / 64
	m := len(s.Rows)

	// Working copy. origIdx lets us report the contradiction row using
	// the caller's original indexing even though Gauss–Jordan reorders
	// rows internally via swaps.
	type workRow struct {
		vars    []uint64
		target  byte
		origIdx int
	}
	work := make([]workRow, m)
	for i, r := range s.Rows {
		v := make([]uint64, words)
		copy(v, r.Vars)
		work[i] = workRow{vars: v, target: r.Target & 1, origIdx: i}
	}

	// pivotCol[k] is the variable index that work[k] pivots on after
	// reduction. Length equals the rank of the system.
	pivotCol := make([]int, 0, s.NumVars)

	// Forward elimination, column-by-column. r tracks the next slot
	// that will receive a pivot row.
	r := 0
	for c := 0; c < s.NumVars; c++ {
		wordIdx := c / 64
		bitMask := uint64(1) << uint(c%64)

		// Find any row at index ≥ r with a 1 in column c.
		pivot := -1
		for i := r; i < m; i++ {
			if work[i].vars[wordIdx]&bitMask != 0 {
				pivot = i
				break
			}
		}
		if pivot == -1 {
			// No pivot for this column — c is a free variable.
			continue
		}

		// Swap the pivot row into position r.
		if pivot != r {
			work[r], work[pivot] = work[pivot], work[r]
		}

		// Gauss–Jordan: clear column c in every *other* row that has
		// a 1 there, above and below the pivot. The row operation is
		// one XOR over the augmented row (Vars and Target together).
		for j := 0; j < m; j++ {
			if j == r {
				continue
			}
			if work[j].vars[wordIdx]&bitMask == 0 {
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

	// Consistency check. Any row reduced to (Vars == 0, Target == 1)
	// is the contradiction 0 = 1. Only non-pivot rows can land in this
	// shape after a complete Gauss–Jordan pass.
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

	// Read off the solution. Output is zero-initialised so that free
	// variables — columns without a pivot — default to 0 implicitly.
	// Only pivot columns may need a 1 bit set.
	bits = make([]byte, (s.NumVars+7)/8)
	for k, col := range pivotCol {
		if work[k].target != 0 {
			bits[col/8] |= 1 << uint(7-col%8)
		}
	}
	return bits, 0, true
}

// SolveBestEffort runs the same Gauss–Jordan elimination as Solve but
// instead of returning false on a contradicting row it silently drops
// that row and continues. Returns the solution bits and the number of
// rows that were dropped.
func (s *System) SolveBestEffort() (bits []byte, dropped int) {
	words := (s.NumVars + 63) / 64
	m := len(s.Rows)

	type workRow struct {
		vars    []uint64
		target  byte
		origIdx int
	}
	work := make([]workRow, m)
	for i, r := range s.Rows {
		v := make([]uint64, words)
		copy(v, r.Vars)
		work[i] = workRow{vars: v, target: r.Target & 1, origIdx: i}
	}

	pivotCol := make([]int, 0, s.NumVars)

	r := 0
	for c := 0; c < s.NumVars; c++ {
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
			if j == r {
				continue
			}
			if work[j].vars[wordIdx]&bitMask == 0 {
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

	// Consistency check: any row reduced to (Vars == 0, Target == 1) is
	// a contradiction. Instead of failing, zero out the target and count.
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

	bits = make([]byte, (s.NumVars+7)/8)
	for k, col := range pivotCol {
		if work[k].target != 0 {
			bits[col/8] |= 1 << uint(7-col%8)
		}
	}
	return bits, dropped
}
