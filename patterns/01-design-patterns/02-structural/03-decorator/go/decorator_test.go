package decorator

import (
	"github.com/xjiang77/rubickx/patterns/support/go/contract"
	"testing"
)

func TestSharedContract(t *testing.T) { contract.Run(t, "../fixtures/contract.json", Evaluate) }
