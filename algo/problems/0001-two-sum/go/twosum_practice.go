//go:build practice

package twosum

// ============================================================
// 练习：Two Sum (LC 1)
// 目标复杂度：时间 O(n)，空间 O(n)
//
// 思路提示（想不起来再看，先自己憋一憋）：
//   1. 一次遍历，用哈希表记录「已见过的值 -> 下标」。
//   2. 每拿到一个 x，先查 target-x 是否已经在表里；命中就返回两个下标。
//   3. 先查再存，否则当 target = 2*x 时会拿自己去配自己。
//
// 跑测试： make practice-go
//     或： go test -tags practice ./problems/0001-two-sum/go/
// 参考解在同目录的 twosum.go —— 卡死了再翻。
// ============================================================

func TwoSumPractice(nums []int, target int) []int {
	// TODO: 你来实现
	return nil
}
