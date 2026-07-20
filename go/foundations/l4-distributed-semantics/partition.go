package distributedsemantics

// Majority returns the smallest strict majority for a cluster. A non-positive
// cluster size has no valid quorum.
func Majority(clusterSize int) int {
	if clusterSize <= 0 {
		return 0
	}
	return clusterSize/2 + 1
}

// QuorumsIntersect reports whether two voter sets share at least one member.
// Duplicate voter IDs inside one set do not change the result.
func QuorumsIntersect(first, second []int) bool {
	members := make(map[int]struct{}, len(first))
	for _, voter := range first {
		members[voter] = struct{}{}
	}
	for _, voter := range second {
		if _, ok := members[voter]; ok {
			return true
		}
	}
	return false
}

// CanCommit models only majority reachability. Voter IDs are one-based and
// must fall within the configured cluster; duplicates are counted once.
func CanCommit(clusterSize int, reachableVoters []int) bool {
	majority := Majority(clusterSize)
	if majority == 0 {
		return false
	}

	unique := make(map[int]struct{}, len(reachableVoters))
	for _, voter := range reachableVoters {
		if voter >= 1 && voter <= clusterSize {
			unique[voter] = struct{}{}
		}
	}
	return len(unique) >= majority
}
