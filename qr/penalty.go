package qr

// Penalty computes the QR mask penalty score for a fully rendered
// symbol grid per ISO/IEC 18004 §7.8.3. Lower is better.
//
// The four penalty rules are:
//
//	N1: For each row/column, every run of 5 or more same-colour
//	    modules contributes 3 + (runLen - 5) points.
//	N2: For every 2×2 block of same-colour modules, 3 points.
//	N3: For every occurrence of the finder-like pattern
//	    1011101_0000 or 0000_1011101 in any row or column, 40 points.
//	N4: For dark-module percentage p%, the penalty is
//	    10 · floor(|p − 50| / 5).
//
// grid is expected to be a square matrix of 0/1 bytes (0 = light,
// 1 = dark) with all function and data modules already resolved.
func Penalty(grid [][]byte) int {
	return penaltyN1(grid) + penaltyN2(grid) + penaltyN3(grid) + penaltyN4(grid)
}

func penaltyN1(grid [][]byte) int {
	n := len(grid)
	score := 0
	add := func(run int) {
		if run >= 5 {
			score += 3 + (run - 5)
		}
	}
	// Rows.
	for r := 0; r < n; r++ {
		run := 1
		for c := 1; c < n; c++ {
			if grid[r][c] == grid[r][c-1] {
				run++
			} else {
				add(run)
				run = 1
			}
		}
		add(run)
	}
	// Columns.
	for c := 0; c < n; c++ {
		run := 1
		for r := 1; r < n; r++ {
			if grid[r][c] == grid[r-1][c] {
				run++
			} else {
				add(run)
				run = 1
			}
		}
		add(run)
	}
	return score
}

func penaltyN2(grid [][]byte) int {
	n := len(grid)
	score := 0
	for r := 0; r < n-1; r++ {
		for c := 0; c < n-1; c++ {
			v := grid[r][c]
			if grid[r][c+1] == v && grid[r+1][c] == v && grid[r+1][c+1] == v {
				score += 3
			}
		}
	}
	return score
}

func penaltyN3(grid [][]byte) int {
	n := len(grid)
	score := 0

	// Pattern A: 1011101 followed by 4 light modules (0000).
	// Pattern B: 4 light modules (0000) followed by 1011101.
	// Combined 11-module patterns.
	patA := [11]byte{1, 0, 1, 1, 1, 0, 1, 0, 0, 0, 0}
	patB := [11]byte{0, 0, 0, 0, 1, 0, 1, 1, 1, 0, 1}

	match := func(line []byte, start int, pat [11]byte) bool {
		for i := 0; i < 11; i++ {
			if line[start+i] != pat[i] {
				return false
			}
		}
		return true
	}

	// Rows.
	row := make([]byte, n)
	for r := 0; r < n; r++ {
		for c := 0; c < n; c++ {
			row[c] = grid[r][c]
		}
		for c := 0; c <= n-11; c++ {
			if match(row, c, patA) || match(row, c, patB) {
				score += 40
			}
		}
	}
	// Columns.
	col := make([]byte, n)
	for c := 0; c < n; c++ {
		for r := 0; r < n; r++ {
			col[r] = grid[r][c]
		}
		for r := 0; r <= n-11; r++ {
			if match(col, r, patA) || match(col, r, patB) {
				score += 40
			}
		}
	}
	return score
}

func penaltyN4(grid [][]byte) int {
	n := len(grid)
	dark := 0
	for r := 0; r < n; r++ {
		for c := 0; c < n; c++ {
			if grid[r][c] == 1 {
				dark++
			}
		}
	}
	total := n * n
	// Percent dark in tenths to avoid floating point.
	pct := dark * 100 / total
	diff := pct - 50
	if diff < 0 {
		diff = -diff
	}
	return 10 * (diff / 5)
}
