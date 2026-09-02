package twosum

// TwoSum 一次遍历 + map：seen[值]=下标，命中 target-x 即返回。O(n)/O(n)。
func TwoSum(nums []int, target int) []int {
	seen := make(map[int]int, len(nums))
	for i, x := range nums {
		if j, ok := seen[target-x]; ok {
			return []int{j, i}
		}
		seen[x] = i
	}
	return nil
}
