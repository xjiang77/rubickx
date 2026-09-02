//go:build practice

package validsudoku

// ============================================================
// 练习：Valid Sudoku (LC 36)
// 目标复杂度：时间 O(81)=O(1)，空间 O(1)
//
// 题目 · LC 36. Valid Sudoku（有效的数独）· Medium
// https://leetcode.cn/problems/valid-sudoku/
//
//   判断一个 9 x 9 的数独是否有效。只需要根据以下规则验证已经填入的数字：
//   1. 每一行的数字 1-9 不能重复；
//   2. 每一列的数字 1-9 不能重复；
//   3. 以粗实线分隔的 9 个 3 x 3 宫内，数字 1-9 不能重复。
//   空白格用 '.' 表示。注意：一个有效的数独（部分已填）不一定是可解的，
//   只需要按上面三条规则验证已填入的数字即可。
//
//   示例：
//     合法棋盘（LC 官方样例）-> true
//     把左上角的 5 改成 8（与同宫的 8 冲突）-> false
//   约束：
//     - board.length == 9
//     - board[i].length == 9
//     - board[i][j] 是一位数字 '1'-'9' 或者 '.'
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
