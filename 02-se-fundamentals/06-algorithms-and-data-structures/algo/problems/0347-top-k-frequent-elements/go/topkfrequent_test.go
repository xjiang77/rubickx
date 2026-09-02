package topkfrequent

import (
	"reflect"
	"sort"
	"testing"
)

func TestTopKFrequent(t *testing.T) {
	cases := []struct {
		nums []int
		k    int
		want []int // 已升序,结果排序后比较
	}{
		{[]int{1, 1, 1, 2, 2, 3}, 2, []int{1, 2}},
		{[]int{1}, 1, []int{1}},
		{[]int{4, 5, 6}, 3, []int{4, 5, 6}},
	}
	for _, c := range cases {
		got := TopKFrequent(c.nums, c.k)
		sort.Ints(got)
		if !reflect.DeepEqual(got, c.want) {
			t.Errorf("TopKFrequent(%v, %d) = %v, want %v", c.nums, c.k, got, c.want)
		}
	}
}
