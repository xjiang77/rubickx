//go:build practice

package validsudoku

import "testing"

// 注意：参考解的测试文件里已经有一个 toBoard，同包内不能重名，这里加 Practice 后缀。
func toBoardPractice(rows []string) [][]byte {
	b := make([][]byte, 9)
	for i, r := range rows {
		b[i] = []byte(r)
	}
	return b
}

var validBoardPractice = []string{
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

// 示例测试：一个合法数独应该返回 true。
func TestIsValidSudokuPractice(t *testing.T) {
	if got := IsValidSudokuPractice(toBoardPractice(validBoardPractice)); !got {
		t.Errorf("IsValidSudokuPractice(合法棋盘) = false, want true")
	}

	// TODO: 三类冲突各写一个用例（把上面的合法棋盘改一个字符就行）
	//   - 同一行出现重复数字
	//   - 同一列出现重复数字
	//   - 同一个 3x3 宫内重复，但行、列都不重复 <- 这条最容易漏
	//
	// TODO: edge case
	//   - 全 '.' 的空棋盘（应该是合法的）
	//   - 只填一个数字
	//   - 重复出现在棋盘最后一格（能不能走到？）
}
