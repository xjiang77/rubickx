package longestconsecutive

// LongestConsecutive set 去重(map[int]struct{}),只从序列起点(x-1 不在集合)向右延伸。
// 每个元素最多访问两次,均摊 O(n)/O(n)。
func LongestConsecutive(nums []int) int {
	set := make(map[int]struct{}, len(nums))
	for _, x := range nums {
		set[x] = struct{}{}
	}
	best := 0
	for x := range set {
		if _, ok := set[x-1]; ok { // 不是起点
			continue
		}
		length := 1
		for {
			if _, ok := set[x+length]; !ok {
				break
			}
			length++
		}
		if length > best {
			best = length
		}
	}
	return best
}
