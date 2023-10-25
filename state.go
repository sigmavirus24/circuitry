package circuitry

import (
	"fmt"
	"time"
)

// CircuitState describes the states of a given CircuitBreaker
type CircuitState uint32

const (
	// CircuitClosed describes a closed CircuitBreaker
	CircuitClosed CircuitState = iota
	// CircuitOpen describes an open CircuitBreaker
	CircuitOpen
	// CircuitHalfOpen describes a half-open CircuitBreaker
	CircuitHalfOpen
)

func (cs CircuitState) String() string {
	switch cs {
	case CircuitClosed:
		return "closed"
	case CircuitOpen:
		return "open"
	case CircuitHalfOpen:
		return "half-open"
	default:
		return fmt.Sprintf("invalid-state: %d", cs)
	}
}

// ExecutionStatus describes the status of a given execution
type ExecutionStatus uint32

const (
	// ExecutionSucceeded describes a successful execution
	ExecutionSucceeded ExecutionStatus = iota
	// ExecutionFailed describes a failed execution
	ExecutionFailed
)

func (es ExecutionStatus) String() string {
	switch es {
	case ExecutionSucceeded:
		return "execution succeeded"
	case ExecutionFailed:
		return "execution failed"
	default:
		return "invalid execution status"
	}
}

// CircuitInformation describes the full state of a given CircuitBreaker to be
// for use with a StorageBackender
type CircuitInformation struct {
	State                CircuitState `json:"state"`
	Generation           uint64       `json:"generation"`
	ConsecutiveFailures  uint64       `json:"consecutive_failures"`
	ConsecutiveSuccesses uint64       `json:"consecutive_successes"`
	Total                uint64       `json:"total"`
	TotalFailures        uint64       `json:"total_failures"`
	TotalSuccesses       uint64       `json:"total_successes"`
	ExpiresAfter         time.Time    `json:"expires_after"`
}

// NewCircuitInformation creates a new CircuitInformation with the state being
// Open and an expiration
func NewCircuitInformation(expiredAfter time.Duration) CircuitInformation {
	return CircuitInformation{
		State:        CircuitClosed,
		ExpiresAfter: time.Now().Add(expiredAfter),
	}
}

type circuitCounts struct {
	ConsecutiveFailures  uint64
	ConsecutiveSuccesses uint64
	Total                uint64
	TotalFailures        uint64
	TotalSuccesses       uint64
}

func (cc *circuitCounts) AddRequest() {
	cc.Total++
}

func (cc *circuitCounts) AddSuccess() {
	cc.ConsecutiveFailures = 0
	cc.ConsecutiveSuccesses++
	cc.TotalSuccesses++
}

func (cc *circuitCounts) AddFailure() {
	cc.ConsecutiveSuccesses = 0
	cc.ConsecutiveFailures++
	cc.TotalFailures++
}

func (cc *circuitCounts) Reset() {
	cc.ConsecutiveFailures = 0
	cc.ConsecutiveSuccesses = 0
	cc.Total = 0
	cc.TotalFailures = 0
	cc.TotalSuccesses = 0
}

func (cc circuitCounts) ToCircuitInformation(generation uint64, state CircuitState, expiry time.Time) CircuitInformation {
	return CircuitInformation{
		State:                state,
		Generation:           generation,
		ConsecutiveFailures:  cc.ConsecutiveFailures,
		ConsecutiveSuccesses: cc.ConsecutiveSuccesses,
		Total:                cc.Total,
		TotalFailures:        cc.TotalFailures,
		TotalSuccesses:       cc.TotalSuccesses,
		ExpiresAfter:         expiry,
	}
}

func fromCircuitInformation(ci CircuitInformation) *circuitCounts {
	return &circuitCounts{
		ConsecutiveFailures:  ci.ConsecutiveFailures,
		ConsecutiveSuccesses: ci.ConsecutiveSuccesses,
		Total:                ci.Total,
		TotalFailures:        ci.TotalFailures,
		TotalSuccesses:       ci.TotalSuccesses,
	}
}
