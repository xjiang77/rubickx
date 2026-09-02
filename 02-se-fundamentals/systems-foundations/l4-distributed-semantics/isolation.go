package distributedsemantics

import "sort"

// TransactionAccess is a deterministic summary of one transaction's logical
// read and write sets. It does not model a database engine or MVCC snapshots.
type TransactionAccess struct {
	TransactionID string
	Reads         []string
	Writes        []string
}

// WriteSkewPair describes a two-transaction rw-antidependency cycle whose
// writes are disjoint and therefore may not trigger a write-write conflict.
type WriteSkewPair struct {
	FirstTransaction       string
	SecondTransaction      string
	FirstReadsSecondWrites []string
	SecondReadsFirstWrites []string
}

// FindWriteSkewPairs finds the narrow two-transaction shape commonly used to
// demonstrate write skew. It is not a general serializability checker.
func FindWriteSkewPairs(transactions []TransactionAccess) []WriteSkewPair {
	pairs := make([]WriteSkewPair, 0)
	for i := 0; i < len(transactions); i++ {
		for j := i + 1; j < len(transactions); j++ {
			first := transactions[i]
			second := transactions[j]

			if len(intersection(first.Writes, second.Writes)) != 0 {
				continue
			}
			firstDependency := intersection(first.Reads, second.Writes)
			secondDependency := intersection(second.Reads, first.Writes)
			if len(firstDependency) == 0 || len(secondDependency) == 0 {
				continue
			}

			pairs = append(pairs, WriteSkewPair{
				FirstTransaction:       first.TransactionID,
				SecondTransaction:      second.TransactionID,
				FirstReadsSecondWrites: firstDependency,
				SecondReadsFirstWrites: secondDependency,
			})
		}
	}
	return pairs
}

func intersection(first, second []string) []string {
	secondSet := make(map[string]struct{}, len(second))
	for _, item := range second {
		secondSet[item] = struct{}{}
	}

	seen := make(map[string]struct{})
	result := make([]string, 0)
	for _, item := range first {
		if _, ok := secondSet[item]; !ok {
			continue
		}
		if _, duplicate := seen[item]; duplicate {
			continue
		}
		seen[item] = struct{}{}
		result = append(result, item)
	}
	sort.Strings(result)
	return result
}
