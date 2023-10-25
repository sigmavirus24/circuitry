package circuitry_test

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/sigmavirus24/circuitry"
)

func TestCircuitStateStringer(t *testing.T) {
	var _ fmt.Stringer = (*circuitry.CircuitState)(nil)
	testCases := []struct {
		name           string
		state          circuitry.CircuitState
		expectedString string
	}{
		{name: "test-closed", state: circuitry.CircuitClosed, expectedString: "closed"},
		{name: "test-half-open", state: circuitry.CircuitHalfOpen, expectedString: "half-open"},
		{name: "test-open", state: circuitry.CircuitOpen, expectedString: "open"},
		{name: "test-invalid", state: circuitry.CircuitState(127), expectedString: "invalid-state: 127"},
	}

	for _, testCase := range testCases {
		tc := testCase
		t.Run(tc.name, func(t *testing.T) {
			if actual := tc.state.String(); actual != tc.expectedString {
				t.Errorf("expected CircuitState(%d).String() to equal '%s' but got '%s'", tc.state, tc.expectedString, actual)
			}
		})
	}
}

func TestExecutionStatusStringer(t *testing.T) {
	var _ fmt.Stringer = (*circuitry.ExecutionStatus)(nil)
	testCases := []struct {
		name           string
		status         circuitry.ExecutionStatus
		expectedString string
	}{
		{"succeeded", circuitry.ExecutionSucceeded, "execution succeeded"},
		{"failed", circuitry.ExecutionFailed, "execution failed"},
		{"invalid", circuitry.ExecutionStatus(127), "invalid execution status"},
	}

	for _, testCase := range testCases {
		tc := testCase
		t.Run(tc.name, func(t *testing.T) {
			if actual := tc.status.String(); actual != tc.expectedString {
				t.Errorf("expected ExecutionStatus(%d).String() to equal '%s' but got '%s'", tc.status, tc.expectedString, actual)
			}
		})
	}
}

func TestNewCircuitInformation(t *testing.T) {
	now := time.Now()
	ci := circuitry.NewCircuitInformation(1 * time.Hour)
	lowerBound := now.Add(59 * time.Minute)
	upperBound := now.Add(61 * time.Minute)
	if ci.ExpiresAfter.Before(lowerBound) || ci.ExpiresAfter.After(upperBound) {
		t.Errorf("expected %s <= ci.ExpiresAfter <= %s but it is %s", lowerBound, upperBound, ci.ExpiresAfter)
	}
	if ci.State != circuitry.CircuitClosed {
		t.Errorf("new CircuitInformation should have closed state but it is %s", ci.State)
	}
}

func TestCircuitInformationJSON(t *testing.T) {
	ci := circuitry.CircuitInformation{
		State:                circuitry.CircuitClosed,
		Generation:           10,
		ConsecutiveFailures:  0,
		ConsecutiveSuccesses: 5,
		Total:                5,
		TotalFailures:        0,
		TotalSuccesses:       5,
		ExpiresAfter:         time.Date(2045, time.December, 1, 23, 59, 59, 0, time.UTC),
	}
	jsonBytes, err := json.Marshal(ci)
	if err != nil {
		t.Fatalf("expected CircuitInformation to Marshal to JSON but got err instead %+q", err)
	}
	unmarshalled := circuitry.CircuitInformation{}
	err = json.Unmarshal(jsonBytes, &unmarshalled)
	if err != nil {
		t.Fatalf("expected CircuitInformation to Unmarshal from JSON but got err instead %+q", err)
	}
	if ci.Generation != unmarshalled.Generation {
		t.Fatalf("CircuitInformation did not survive round-trip through JSON %+q != %+q", ci, unmarshalled)
	}
}
