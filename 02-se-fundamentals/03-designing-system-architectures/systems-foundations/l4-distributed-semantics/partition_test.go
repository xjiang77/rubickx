package distributedsemantics

import "testing"

func TestMajority(t *testing.T) {
	tests := []struct {
		clusterSize int
		want        int
	}{
		{clusterSize: 0, want: 0},
		{clusterSize: 1, want: 1},
		{clusterSize: 3, want: 2},
		{clusterSize: 5, want: 3},
	}

	for _, test := range tests {
		if got := Majority(test.clusterSize); got != test.want {
			t.Errorf("Majority(%d) = %d, want %d", test.clusterSize, got, test.want)
		}
	}
}

func TestQuorumsIntersect(t *testing.T) {
	if !QuorumsIntersect([]int{1, 2}, []int{2, 3}) {
		t.Fatal("3-voter majorities should intersect")
	}
	if !QuorumsIntersect([]int{1, 2, 3}, []int{3, 4, 5}) {
		t.Fatal("5-voter majorities should intersect")
	}
	if QuorumsIntersect([]int{1, 2}, []int{3, 4}) {
		t.Fatal("disjoint minority sets should not intersect")
	}
}

func TestCanCommitThreeVoterPartition(t *testing.T) {
	if !CanCommit(3, []int{1, 2}) {
		t.Fatal("2-voter side of a 2+1 partition should have a majority")
	}
	if CanCommit(3, []int{3}) {
		t.Fatal("1-voter side of a 2+1 partition should not have a majority")
	}
}

func TestCanCommitFiveVoterPartition(t *testing.T) {
	if !CanCommit(5, []int{1, 2, 3}) {
		t.Fatal("3-voter side of a 3+2 partition should have a majority")
	}
	if CanCommit(5, []int{4, 5}) {
		t.Fatal("2-voter side of a 3+2 partition should not have a majority")
	}
}

func TestCanCommitDeduplicatesAndValidatesVoters(t *testing.T) {
	if CanCommit(3, []int{1, 1, 4}) {
		t.Fatal("duplicate or out-of-range IDs must not create a majority")
	}
	if !CanCommit(3, []int{1, 2, 2}) {
		t.Fatal("two unique valid voters should create a majority")
	}
}
