package topkfrequent

// TopKFrequent 计数 + 桶排序:下标=频次,从高频桶向低频收集 k 个。O(n)/O(n)。
func TopKFrequent(nums []int, k int) []int {
	counts := make(map[int]int, len(nums))
	for _, x := range nums {
		counts[x]++
	}
	buckets := make([][]int, len(nums)+1)
	for x, c := range counts {
		buckets[c] = append(buckets[c], x)
	}
	res := make([]int, 0, k)
	for c := len(buckets) - 1; c > 0 && len(res) < k; c-- {
		res = append(res, buckets[c]...)
	}
	return res[:k]
}
