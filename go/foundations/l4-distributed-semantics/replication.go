package distributedsemantics

// ReplicaRead records the version required by a caller and the version
// observed from one replica. Versions are supplied by the system under test.
type ReplicaRead struct {
	OperationID     string
	ReplicaID       string
	RequiredVersion uint64
	ObservedVersion uint64
}

// ReplicaFreshnessViolation is a counterexample to a minimum-version read.
type ReplicaFreshnessViolation struct {
	ReadIndex       int
	OperationID     string
	ReplicaID       string
	RequiredVersion uint64
	ObservedVersion uint64
}

// CheckReplicaFreshness reports reads served below the caller's required
// version. It does not infer how versions are replicated or ordered.
func CheckReplicaFreshness(reads []ReplicaRead) []ReplicaFreshnessViolation {
	violations := make([]ReplicaFreshnessViolation, 0)
	for i, read := range reads {
		if read.ObservedVersion >= read.RequiredVersion {
			continue
		}
		violations = append(violations, ReplicaFreshnessViolation{
			ReadIndex:       i,
			OperationID:     read.OperationID,
			ReplicaID:       read.ReplicaID,
			RequiredVersion: read.RequiredVersion,
			ObservedVersion: read.ObservedVersion,
		})
	}
	return violations
}
