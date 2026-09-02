package containsduplicate

import "testing"

func TestContainsDuplicate(t *testing.T) {
	cases := []struct {
		nums []int
		want bool
	}{
		{[]int{1, 2, 3, 1}, true},
		{[]int{1, 2, 3, 4}, false},
		{[]int{}, false},
	}
	for _, c := range cases {
		if got := ContainsDuplicate(c.nums); got != c.want {
			t.Errorf("ContainsDuplicate(%v) = %v, want %v", c.nums, got, c.want)
		}
	}
}
