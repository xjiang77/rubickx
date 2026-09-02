//go:build practice

package topkfrequent

// ============================================================
// 练习：Top K Frequent Elements (LC 347)
// 目标复杂度：时间 O(n)，空间 O(n)
//
// 题目 · LC 347. Top K Frequent Elements（前 K 个高频元素）· Medium
// https://leetcode.cn/problems/top-k-frequent-elements/
//
//   给定一个整数数组 nums 和一个整数 k，返回其中出现频率前 k 高的元素。
//   可以按任意顺序返回答案。
//
//   示例：
//     nums = [1,1,1,2,2,3], k = 2   ->  [1,2]
//     nums = [1],           k = 1   ->  [1]
//   约束：
//     - 1 <= nums.length <= 10^5
//     - -10^4 <= nums[i] <= 10^4
//     - k 的取值范围是 [1, 数组中不相同的元素的个数]
//     - 题目数据保证答案唯一
//   进阶：所设计算法的时间复杂度必须优于 O(n log n)。
//
// 思路提示（想不起来再看，先自己憋一憋）：
//   1. 第一步永远是计数：map[值] -> 出现次数。
//   2. 堆的写法是 O(n log k)；桶排序能做到 O(n)。
//   3. 桶排序思路：出现次数最多不超过 n，所以开 n+1 个桶，buckets[c] 放所有出现 c 次的值。
//   4. 从高频桶往低频桶倒着扫，收满 k 个就停。
//   5. 结果顺序题目不做要求，所以测试里要先排序再比较。
//
// 跑测试： make practice-go
//     或： go test -tags practice ./problems/0347-top-k-frequent-elements/go/
// 参考解在同目录的 topkfrequent.go —— 卡死了再翻。
// ============================================================

func TopKFrequentPractice(nums []int, k int) []int {
	// 1. 计数
	freqMap := make(map[int]int, len(nums))
	for _, num := range nums {
		freqMap[num]++
	}
	// 2. 桶排序
	buckets := make([][]int, len(nums)+1)
	for num, freq := range freqMap {
		buckets[freq] = append(buckets[freq], num)
	}
	// 3. 从高频桶往低频桶倒着扫，收满 k 个就停
	res := make([]int, 0, k)
	for i := len(buckets) - 1; i >= 0 && len(res) < k; i-- {
		res = append(res, buckets[i]...)
	}
	return res[:k]
}
