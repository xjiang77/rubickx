package validanagram

import "testing"

func TestIsAnagram(t *testing.T) {
	cases := []struct {
		s, t string
		want bool
	}{
		{"anagram", "nagaram", true},
		{"rat", "car", false},
		{"a", "ab", false},
	}
	for _, c := range cases {
		if got := IsAnagram(c.s, c.t); got != c.want {
			t.Errorf("IsAnagram(%q, %q) = %v, want %v", c.s, c.t, got, c.want)
		}
	}
}
