package distributedsemantics

import "errors"

var ErrMissingMessageID = errors.New("delivery message ID is required for idempotent application")
var ErrConflictingDelivery = errors.New("duplicate message ID has a conflicting effect")

// Delivery represents an at-least-once message and its business effect.
type Delivery struct {
	MessageID string
	Effect    int
}

// ApplyNaive applies every delivery, including duplicates.
func ApplyNaive(deliveries []Delivery) int {
	total := 0
	for _, delivery := range deliveries {
		total += delivery.Effect
	}
	return total
}

// ApplyIdempotent applies the first delivery for each stable message ID. It
// fails explicitly when an ID is missing because deduplication would be unsafe.
func ApplyIdempotent(deliveries []Delivery) (int, error) {
	for _, delivery := range deliveries {
		if delivery.MessageID == "" {
			return 0, ErrMissingMessageID
		}
	}

	seen := make(map[string]int, len(deliveries))
	total := 0
	for _, delivery := range deliveries {
		if previous, duplicate := seen[delivery.MessageID]; duplicate {
			if previous != delivery.Effect {
				return 0, ErrConflictingDelivery
			}
			continue
		}
		seen[delivery.MessageID] = delivery.Effect
		total += delivery.Effect
	}
	return total, nil
}
