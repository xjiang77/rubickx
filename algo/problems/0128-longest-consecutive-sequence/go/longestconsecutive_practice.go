//go:build practice

package longestconsecutive

// ============================================================
// 练习：Longest Consecutive Sequence (LC 128)
// 目标复杂度：时间 O(n)，空间 O(n)
//
// 思路提示（想不起来再看，先自己憋一憋）：
//   1. 先把所有数塞进一个 set，查询 O(1)。
//   2. 关键剪枝：只在 x 是一段连续序列的「起点」时才展开，即 x-1 不在 set 里。
//   3. 从起点开始 x+1, x+2, ... 一直数到断掉，记录最长长度。
//   4. 有了这个剪枝，每个元素最多被展开访问一次，整体才是 O(n)；否则会退化成 O(n²)。
//   5. 空数组要返回 0。
//
// 跑测试： make practice-go
//     或： go test -tags practice ./problems/0128-longest-consecutive-sequence/go/
// 参考解在同目录的 longestconsecutive.go —— 卡死了再翻。
// ============================================================

func LongestConsecutivePractice(nums []int) int {
	// TODO: 你来实现
	return 0
}
