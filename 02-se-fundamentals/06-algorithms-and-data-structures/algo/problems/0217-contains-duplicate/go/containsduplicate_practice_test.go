//go:build practice

package containsduplicate

import "testing"

// 示例测试。
func TestContainsDuplicatePractice(t *testing.T) {
	nums := []int{1, 2, 3, 1}
	if got := ContainsDuplicatePractice(nums); !got {
		t.Errorf("ContainsDuplicatePractice(%v) = false, want true", nums)
	}

	nums = []int{1, 2, 3, 4}
	if got := ContainsDuplicatePractice(nums); got {
		t.Errorf("ContainsDuplicatePractice(%v) = true, want false", nums)
	}

	nums = []int{1, 1}
	if got := ContainsDuplicatePractice(nums); !got {
		t.Errorf("ContainsDuplicatePractice(%v) = false, want true", nums)
	}

	nums = []int{}
	if got := ContainsDuplicatePractice(nums); got {
		t.Errorf("ContainsDuplicatePractice(%v) = true, want false", nums)
	}

	nums = []int{1}
	if got := ContainsDuplicatePractice(nums); got {
		t.Errorf("ContainsDuplicatePractice(%v) = true, want false", nums)
	}

	nums = []int{-1, -2, -3, -4}
	if got := ContainsDuplicatePractice(nums); got {
		t.Errorf("ContainsDuplicatePractice(%v) = true, want false", nums)
	}

	nums = []int{0, 0, 0, 0}
	if got := ContainsDuplicatePractice(nums); !got {
		t.Errorf("ContainsDuplicatePractice(%v) = false, want true", nums)
	}

	nums = []int{1, 1, 1, 1}
	if got := ContainsDuplicatePractice(nums); !got {
		t.Errorf("ContainsDuplicatePractice(%v) = false, want true", nums)
	}

	nums = []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}
	if got := ContainsDuplicatePractice(nums); got {
		t.Errorf("ContainsDuplicatePractice(%v) = true, want false", nums)
	}

	nums = []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 1}
	if got := ContainsDuplicatePractice(nums); !got {
		t.Errorf("ContainsDuplicatePractice(%v) = false, want true", nums)
	}
	// TODO: 补充更多场景
	//   - 全不重复：[1,2,3,4] -> false
	//   - 相邻重复：[1,1] -> true
	//   - 重复出现在数组最后两个位置（确认你没提前 return 漏掉）
	//
	// TODO: edge case
	//   - 空数组 -> false
	//   - 单元素 -> false
	//   - 含负数、含 0
	//   - 全是同一个值的长数组
}
