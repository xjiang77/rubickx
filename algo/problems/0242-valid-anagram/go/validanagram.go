package validanagram

// IsAnagram 用计数 map 统计 s，再用 t 抵消；任一字符计数变负即非异位词。O(n)/O(k)。
// 注意：len 比的是字节数，按 rune 遍历可正确处理多字节字符。
func IsAnagram(s, t string) bool {
	if len(s) != len(t) {
		return false
	}
	count := make(map[rune]int)
	for _, c := range s {
		count[c]++
	}
	for _, c := range t {
		count[c]--
		if count[c] < 0 {
			return false
		}
	}
	return true
}
