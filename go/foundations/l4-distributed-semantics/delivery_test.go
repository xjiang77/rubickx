package distributedsemantics

import (
	"errors"
	"testing"
)

func TestApplyNaiveDuplicatesEffect(t *testing.T) {
	deliveries := []Delivery{
		{MessageID: "payment-1", Effect: 100},
		{MessageID: "payment-1", Effect: 100},
	}

	if got, want := ApplyNaive(deliveries), 200; got != want {
		t.Fatalf("ApplyNaive() = %d, want %d", got, want)
	}
}

func TestApplyIdempotentAppliesFirstDeliveryOnce(t *testing.T) {
	deliveries := []Delivery{
		{MessageID: "payment-1", Effect: 100},
		{MessageID: "payment-1", Effect: 100},
		{MessageID: "refund-1", Effect: -20},
	}

	got, err := ApplyIdempotent(deliveries)
	if err != nil {
		t.Fatalf("ApplyIdempotent() error = %v", err)
	}
	if want := 80; got != want {
		t.Fatalf("ApplyIdempotent() = %d, want %d", got, want)
	}
}

func TestApplyIdempotentRejectsMissingIdentity(t *testing.T) {
	_, err := ApplyIdempotent([]Delivery{{MessageID: "", Effect: 100}})
	if !errors.Is(err, ErrMissingMessageID) {
		t.Fatalf("ApplyIdempotent() error = %v, want %v", err, ErrMissingMessageID)
	}
}

func TestApplyIdempotentRejectsConflictingDuplicate(t *testing.T) {
	_, err := ApplyIdempotent([]Delivery{
		{MessageID: "payment-1", Effect: 100},
		{MessageID: "payment-1", Effect: 200},
	})
	if !errors.Is(err, ErrConflictingDelivery) {
		t.Fatalf("ApplyIdempotent() error = %v, want %v", err, ErrConflictingDelivery)
	}
}
