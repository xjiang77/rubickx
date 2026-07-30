//go:build practice

package twosum

import (
	"reflect"
	"testing"
)

// 示例测试：先让这一个跑通，再照着往下补自己的场景。
func TestTwoSumPractice(t *testing.T) {
	nums, target := []int{2, 7, 11, 15}, 9
	want := []int{0, 1}
	if got := TwoSumPractice(nums, target); !reflect.DeepEqual(got, want) {
		t.Errorf("TwoSumPractice(%v, %d) = %v, want %v", nums, target, got, want)
	}

	// TODO: 补充更多正常场景
	//   - 答案落在数组中间：[3,2,4] target=6
	//   - 两个值相同的数：[3,3] target=6
	//   - 长一点的数组，答案在末尾
	//
	// TODO: 想清楚这些 edge case 要不要测、期望是什么
	//   - 负数 / 0 参与求和
	//   - 无解时你的实现返回什么？（nil？空切片？）测试应该把这个约定钉死
	//   - 返回的下标顺序有没有要求
}
