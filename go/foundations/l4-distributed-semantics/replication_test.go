package distributedsemantics

import "testing"

func TestCheckReplicaFreshness(t *testing.T) {
	reads := []ReplicaRead{
		{OperationID: "route-read-1", ReplicaID: "follower-a", RequiredVersion: 43, ObservedVersion: 42},
		{OperationID: "route-read-2", ReplicaID: "leader", RequiredVersion: 43, ObservedVersion: 43},
		{OperationID: "route-read-3", ReplicaID: "follower-b", RequiredVersion: 43, ObservedVersion: 44},
	}

	violations := CheckReplicaFreshness(reads)
	if len(violations) != 1 {
		t.Fatalf("violations = %v, want exactly one", violations)
	}
	got := violations[0]
	if got.ReadIndex != 0 || got.OperationID != "route-read-1" || got.ReplicaID != "follower-a" || got.RequiredVersion != 43 || got.ObservedVersion != 42 {
		t.Fatalf("violation = %+v, want stale follower-a read at index 0", got)
	}
}

func TestCheckReplicaFreshnessAllowsNoMinimumVersion(t *testing.T) {
	reads := []ReplicaRead{{OperationID: "analytics-read", ReplicaID: "regional", ObservedVersion: 17}}
	if violations := CheckReplicaFreshness(reads); len(violations) != 0 {
		t.Fatalf("violations = %v, want none", violations)
	}
}
