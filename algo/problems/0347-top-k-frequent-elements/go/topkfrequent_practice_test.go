//go:build practice

package topkfrequent

import (
	"reflect"
	"sort"
	"testing"
)

// 返回顺序题目不做要求，比较前先排序。
// 示例测试：[1,1,1,2,2,3], k=2 -> {1,2}。
func TestTopKFrequentPractice(t *testing.T) {
	nums, k := []int{1, 1, 1, 2, 2, 3}, 2
	got := TopKFrequentPractice(nums, k)
	sort.Ints(got)
	want := []int{1, 2}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("TopKFrequentPractice(%v, %d) = %v, want %v", nums, k, got, want)
	}

	// TODO: 补充更多场景
	//   - k=1，只取最高频
	//   - k 等于不同元素的个数（等于全取）
	//   - 所有元素频次都一样，比如 [1,2,3] k=2（答案不唯一，测试怎么写才稳？）
	//
	// TODO: edge case
	//   - 单元素数组 [1], k=1
	//   - 含负数
	//   - 频次并列卡在第 k 名边界上，比如 [1,1,2,2,3] k=2
	//   - 检查返回切片长度必须恰好是 k（桶排序写法很容易多返回几个）
}
