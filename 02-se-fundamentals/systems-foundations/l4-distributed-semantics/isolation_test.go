package distributedsemantics

import "testing"

func TestFindWriteSkewPairs(t *testing.T) {
	transactions := []TransactionAccess{
		{TransactionID: "alice-off-call", Reads: []string{"alice", "bob"}, Writes: []string{"alice"}},
		{TransactionID: "bob-off-call", Reads: []string{"alice", "bob"}, Writes: []string{"bob"}},
	}

	pairs := FindWriteSkewPairs(transactions)
	if len(pairs) != 1 {
		t.Fatalf("pairs = %v, want exactly one", pairs)
	}
	got := pairs[0]
	if got.FirstTransaction != "alice-off-call" || got.SecondTransaction != "bob-off-call" {
		t.Fatalf("pair = %+v, want alice/bob transactions", got)
	}
	if len(got.FirstReadsSecondWrites) != 1 || got.FirstReadsSecondWrites[0] != "bob" {
		t.Fatalf("first dependency = %v, want [bob]", got.FirstReadsSecondWrites)
	}
	if len(got.SecondReadsFirstWrites) != 1 || got.SecondReadsFirstWrites[0] != "alice" {
		t.Fatalf("second dependency = %v, want [alice]", got.SecondReadsFirstWrites)
	}
}

func TestFindWriteSkewPairsExcludesWriteWriteConflict(t *testing.T) {
	transactions := []TransactionAccess{
		{TransactionID: "increment-a", Reads: []string{"counter"}, Writes: []string{"counter"}},
		{TransactionID: "increment-b", Reads: []string{"counter"}, Writes: []string{"counter"}},
	}

	if pairs := FindWriteSkewPairs(transactions); len(pairs) != 0 {
		t.Fatalf("pairs = %v, want none for a direct write-write conflict", pairs)
	}
}

func TestFindWriteSkewPairsRequiresMutualDependency(t *testing.T) {
	transactions := []TransactionAccess{
		{TransactionID: "reader", Reads: []string{"quota"}},
		{TransactionID: "writer", Writes: []string{"quota"}},
	}

	if pairs := FindWriteSkewPairs(transactions); len(pairs) != 0 {
		t.Fatalf("pairs = %v, want none for a one-way dependency", pairs)
	}
}
