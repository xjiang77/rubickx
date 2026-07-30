//go:build practice

package productexceptself

import (
	"reflect"
	"testing"
)

// 示例测试：[1,2,3,4] -> [24,12,8,6]。
func TestProductExceptSelfPractice(t *testing.T) {
	nums := []int{1, 2, 3, 4}
	want := []int{24, 12, 8, 6}
	if got := ProductExceptSelfPractice(nums); !reflect.DeepEqual(got, want) {
		t.Errorf("ProductExceptSelfPractice(%v) = %v, want %v", nums, got, want)
	}

	// TODO: 补充更多场景
	//   - 含一个 0：[-1,1,0,-3,3] -> [0,0,9,0,0]
	//   - 含两个 0：结果应该全是 0
	//   - 全负数、正负混合（注意符号）
	//
	// TODO: edge case
	//   - 长度 2 的数组：[a,b] -> [b,a]
	//   - 含 1 的数组
	//   - 想想溢出：如果元素很大，int 会不会溢出？Go 的 int 是 64 位，Java 的 int 是 32 位，两边行为一样吗？
}
