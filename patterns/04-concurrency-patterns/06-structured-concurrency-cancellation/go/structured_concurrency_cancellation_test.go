package structuredconcurrencycancellation

import (
	"sync"
	"testing"

	"github.com/xjiang77/rubickx/patterns/support/go/contract"
)

func TestSharedContract(t *testing.T) {
	contract.Run(t, "../fixtures/contract.json", Evaluate)
}

func TestFailureCancelsSiblingAndAllGoroutinesJoin(t *testing.T) {
	start := make(chan struct{})
	cancelled := make(chan struct{})
	events := []string{}
	var lock sync.Mutex
	var workers sync.WaitGroup
	workers.Add(2)

	go func() {
		defer workers.Done()
		<-start
		lock.Lock()
		events = append(events, "failed")
		lock.Unlock()
		close(cancelled)
	}()
	go func() {
		defer workers.Done()
		<-start
		<-cancelled
		lock.Lock()
		events = append(events, "cancelled")
		lock.Unlock()
	}()

	close(start)
	workers.Wait()
	if len(events) != 2 || events[0] != "failed" || events[1] != "cancelled" {
		t.Fatalf("unexpected terminal events: %v", events)
	}
}
