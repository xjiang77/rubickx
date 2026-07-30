package validsudoku

// IsValidSudoku 三组位掩码(rows/cols/boxes 各 9 个 uint16),bit d 表示数字 d+1 已出现。
// box 下标 = (r/3)*3 + c/3。一次扫描,重复即 false。O(81) 时间,O(1) 空间。
func IsValidSudoku(board [][]byte) bool {
	var rows, cols, boxes [9]uint16
	for r := 0; r < 9; r++ {
		for c := 0; c < 9; c++ {
			ch := board[r][c]
			if ch == '.' {
				continue
			}
			bit := uint16(1) << (ch - '1')
			b := (r/3)*3 + c/3
			if rows[r]&bit != 0 || cols[c]&bit != 0 || boxes[b]&bit != 0 {
				return false
			}
			rows[r] |= bit
			cols[c] |= bit
			boxes[b] |= bit
		}
	}
	return true
}
