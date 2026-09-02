//go:build practice

package groupanagrams

import (
	"reflect"
	"sort"
	"strings"
	"testing"
)

// 分组顺序和组内顺序都不做要求，比较前先归一化。
// 参考解的测试里已有 normalize，这里换个名字避免同包冲突。
func normalizePractice(groups [][]string) [][]string {
	out := make([][]string, len(groups))
	for i, g := range groups {
		cp := append([]string(nil), g...)
		sort.Strings(cp)
		out[i] = cp
	}
	sort.Slice(out, func(i, j int) bool {
		return strings.Join(out[i], ",") < strings.Join(out[j], ",")
	})
	return out
}

// 示例测试。
func TestGroupAnagramsPractice(t *testing.T) {
	got := normalizePractice(GroupAnagramsPractice([]string{"eat", "tea", "tan", "ate", "nat", "bat"}))
	want := normalizePractice([][]string{{"bat"}, {"nat", "tan"}, {"ate", "eat", "tea"}})
	if !reflect.DeepEqual(got, want) {
		t.Errorf("GroupAnagramsPractice = %v, want %v", got, want)
	}

	// TODO: 补充更多场景
	//   - 每个词自成一组（没有任何异位词）
	//   - 所有词属于同一组
	//   - 有重复的输入词，比如 ["ab","ab"]，它们应该在同一组且都保留
	//
	// TODO: edge case
	//   - 空输入 []string{}
	//   - 含空字符串 ""
	//   - 单字符 ["a"]
	//   - 长度相同但字符不同的词，会不会被你的 key 误判成一组？
}
