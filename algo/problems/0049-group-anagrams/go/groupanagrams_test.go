package groupanagrams

import (
	"reflect"
	"sort"
	"strings"
	"testing"
)

func normalize(groups [][]string) [][]string {
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

func TestGroupAnagrams(t *testing.T) {
	got := normalize(GroupAnagrams([]string{"eat", "tea", "tan", "ate", "nat", "bat"}))
	want := normalize([][]string{{"bat"}, {"nat", "tan"}, {"ate", "eat", "tea"}})
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}
