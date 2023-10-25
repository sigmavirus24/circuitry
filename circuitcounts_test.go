package circuitry

import (
	"testing"
	"time"
)

func TestCircuitCountsAddRequest(t *testing.T) {
	cc := circuitCounts{}
	cc.AddRequest()
	if cc.Total != 1 {
		t.Fatalf("expected Total to be 1 after a new request but got %d", cc.Total)
	}
	cc.Reset()
	if cc.ConsecutiveSuccesses != 0 {
		t.Fatalf("expected ConsecutiveSuccesses to be 0 after a reset but got %d", cc.ConsecutiveSuccesses)
	}
}

func TestCircuitCountsAddSuccess(t *testing.T) {
	cc := circuitCounts{ConsecutiveFailures: 2}
	cc.AddSuccess()
	if cc.TotalSuccesses != 1 {
		t.Fatalf("expected TotalSuccesses to be 1 after a success but got %d", cc.TotalSuccesses)
	}
	if cc.ConsecutiveSuccesses != 1 {
		t.Fatalf("expected ConsecutiveSuccesses to be 1 after a success but got %d", cc.ConsecutiveSuccesses)
	}
	if cc.ConsecutiveFailures != 0 {
		t.Fatalf("expected ConsecutiveFailures to be 0 after a success but got %d", cc.ConsecutiveFailures)
	}
}

func TestCircuitCountsAddFailure(t *testing.T) {
	cc := circuitCounts{ConsecutiveSuccesses: 2}
	cc.AddFailure()
	if cc.TotalFailures != 1 {
		t.Fatalf("expected TotalFailures to be 1 after a failure but got %d", cc.TotalFailures)
	}
	if cc.ConsecutiveFailures != 1 {
		t.Fatalf("expected ConsecutiveFailures to be 1 after a failure but got %d", cc.ConsecutiveFailures)
	}
	if cc.ConsecutiveSuccesses != 0 {
		t.Fatalf("expected ConsecutiveSuccesses to be 0 after a failure but got %d", cc.ConsecutiveSuccesses)
	}
}

func TestCircuitCountsReset(t *testing.T) {
	// NOTE: this count state shouldn't be possible, just want non-zero values
	// everywhere
	cc := circuitCounts{ConsecutiveFailures: 2, ConsecutiveSuccesses: 2, Total: 2, TotalFailures: 2, TotalSuccesses: 2}
	cc.Reset()
	if cc.ConsecutiveFailures != 0 || cc.ConsecutiveSuccesses != 0 || cc.Total != 0 || cc.TotalFailures != 0 || cc.TotalSuccesses != 0 {
		t.Errorf("circuitCounts(%+q).Reset() did not reset the counts", cc)
	}
}

func TestCircuitCountsToCircuitInformation(t *testing.T) {
	cc := circuitCounts{ConsecutiveFailures: 0, ConsecutiveSuccesses: 5, Total: 15, TotalFailures: 1, TotalSuccesses: 14}
	ci := cc.ToCircuitInformation(1, CircuitClosed, time.Date(2155, time.December, 1, 23, 59, 59, 0, time.UTC))
	if ci.Total != 15 {
		t.Fatalf("converting from circuitCounts(%+q) to CircuitInformation(%+q) failed", cc, ci)
	}
}
