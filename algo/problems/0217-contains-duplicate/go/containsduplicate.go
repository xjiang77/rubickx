package containsduplicate

// ContainsDuplicate 用 set（map[int]struct{}）边遍历边查。O(n)/O(n)。
func ContainsDuplicate(nums []int) bool {
	seen := make(map[int]struct{}, len(nums))
	for _, x := range nums {
		if _, ok := seen[x]; ok {
			return true
		}
		seen[x] = struct{}{}
	}
	return false
}
