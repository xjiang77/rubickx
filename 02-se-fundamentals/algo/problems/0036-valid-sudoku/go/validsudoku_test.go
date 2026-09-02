package validsudoku

import "testing"

var valid = []string{
	"53..7....",
	"6..195...",
	".98....6.",
	"8...6...3",
	"4..8.3..1",
	"7...2...6",
	".6....28.",
	"...419..5",
	"....8..79",
}

func toBoard(rows []string) [][]byte {
	board := make([][]byte, len(rows))
	for i, row := range rows {
		board[i] = []byte(row)
	}
	return board
}

func TestValid(t *testing.T) {
	if !IsValidSudoku(toBoard(valid)) {
		t.Error("expected valid board to pass")
	}
}

func TestBoxDuplicate(t *testing.T) {
	rows := append([]string{"8" + valid[0][1:]}, valid[1:]...) // (0,0) 5→8,与 (2,2) 的 8 同 box
	if IsValidSudoku(toBoard(rows)) {
		t.Error("expected box duplicate to fail")
	}
}

func TestRowDuplicate(t *testing.T) {
	rows := append([]string{valid[0][:2] + "5" + valid[0][3:]}, valid[1:]...) // 第 0 行两个 5
	if IsValidSudoku(toBoard(rows)) {
		t.Error("expected row duplicate to fail")
	}
}
