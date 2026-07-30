package longestconsecutive

import "testing"

func TestLongestConsecutive(t *testing.T) {
	cases := []struct {
		nums []int
		want int
	}{
		{[]int{100, 4, 200, 1, 3, 2}, 4},
		{[]int{0, 3, 7, 2, 5, 8, 4, 6, 0, 1}, 9},
		{[]int{}, 0},
		{[]int{5}, 1},
	}
	for _, c := range cases {
		if got := LongestConsecutive(c.nums); got != c.want {
			t.Errorf("LongestConsecutive(%v) = %d, want %d", c.nums, got, c.want)
		}
	}
}
