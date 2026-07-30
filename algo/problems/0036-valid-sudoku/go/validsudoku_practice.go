//go:build practice

package validsudoku

// ============================================================
// 练习：Valid Sudoku (LC 36)
// 目标复杂度：时间 O(81)=O(1)，空间 O(1)
//
// 思路提示（想不起来再看，先自己憋一憋）：
//   1. 三类约束各自维护一份「已出现过的数字」：9 行、9 列、9 个 3x3 宫。
//   2. 宫的下标公式：b = (r/3)*3 + c/3。
//   3. '.' 直接跳过；只要任一类里出现重复就立刻返回 false。
//   4. 「已出现过」可以用 set，也可以用位掩码（每格一个 uint16 / int，第 d 位表示数字 d 出现过）。
//
// 跑测试： make practice-go
//     或： go test -tags practice ./problems/0036-valid-sudoku/go/
// 参考解在同目录的 validsudoku.go —— 卡死了再翻。
// ============================================================

func IsValidSudokuPractice(board [][]byte) bool {
	// TODO: 你来实现
	return false
}
