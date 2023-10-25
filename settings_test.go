package circuitry_test

import (
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/sigmavirus24/circuitry"
	"github.com/sigmavirus24/circuitry/backends"
	"github.com/sigmavirus24/circuitry/log"
)

func TestDefaultNameFunc(t *testing.T) {
	expected := "identical"
	if output := circuitry.DefaultNameFunc(expected, map[string]any{}); expected != output {
		t.Fatalf("expected DefaultNamer to not modify the input, but got %s", output)
	}
}

type myError struct{ expected bool }

func (m myError) Error() string {
	return "test error"
}
func (m myError) IsExpected() bool {
	return m.expected
}

func TestDefaultErrorMatcher(t *testing.T) {
	testCases := map[string]struct {
		input          error
		expectedOutput circuitry.ExecutionStatus
	}{
		"succeeded without error":             {nil, circuitry.ExecutionSucceeded},
		"failed with error":                   {fmt.Errorf("test error"), circuitry.ExecutionFailed},
		"is expected interface true":          {myError{true}, circuitry.ExecutionSucceeded},
		"is expected interface false":         {myError{false}, circuitry.ExecutionFailed},
		"wrapped with ExpectedConditionError": {circuitry.WrapExpectedConditionError(fmt.Errorf("test error")), circuitry.ExecutionSucceeded},
	}

	for name, testCase := range testCases {
		tc := testCase
		t.Run(name, func(t *testing.T) {
			actualOutput := circuitry.DefaultErrorMatcher(tc.input)
			if actualOutput != tc.expectedOutput {
				t.Errorf("DefaultErrorMatcher(%v) got %s; expected %s", tc.input, actualOutput, tc.expectedOutput)
			}
		})
	}
}

func TestNewFactorySettingsNoOptions(t *testing.T) {
	s, err := circuitry.NewFactorySettings()
	if err != nil {
		t.Fatalf("expected default settings to result in an error but got %v", err)
	}
	expected := "identical"
	if output := s.GenerateName(expected, map[string]any{}); expected != output {
		t.Fatalf("expected default settings to not modify the input, but got %s", output)
	}
}

func TestFactorySettingsGenerateName(t *testing.T) {
	s, err := circuitry.NewFactorySettings(circuitry.WithNameFunc(func(circuit string, circuitContext map[string]any) string {
		if appendToName, ok := circuitContext["append"].(string); ok {
			return fmt.Sprintf("%s:%s", circuit, appendToName)
		}
		return circuit
	}))
	if err != nil {
		t.Fatalf("expected valid settings but got error %v", err)
	}
	testCases := map[string]struct {
		name         string
		appendToName string
		expected     string
	}{
		"name only":         {"circuit", "", "circuit"},
		"name from context": {"circuit", "append-me", "circuit:append-me"},
	}
	for name, testCase := range testCases {
		tc := testCase
		t.Run(name, func(t *testing.T) {
			circuitCtx := map[string]any{}
			if tc.appendToName != "" {
				circuitCtx["append"] = tc.appendToName
			}
			if output := s.GenerateName(tc.name, circuitCtx); tc.expected != output {
				t.Errorf("expected custom GenerateName(%s, %+v) = %s; got %s", tc.name, circuitCtx, tc.expected, output)
			}
		})
	}
}

func TestBuildFactorySettingsWithConflictingOptions(t *testing.T) {
	testCases := map[string]struct {
		opt             circuitry.SettingsOption
		expectedErr     error
		expectedErrName string
	}{
		"storage backend conflict": {
			backends.WithInMemoryBackend(),
			circuitry.ErrStorageBackendAlreadySet,
			"ErrStorageBackendAlreadySet",
		},
		"fallback matcher conflict": {
			circuitry.WithDefaultFallbackErrorMatcher(),
			circuitry.ErrFallbackErrorMatcherAlreadySet,
			"ErrFallbackMatcherAlreadySet",
		},
		"name function conflict": {
			circuitry.WithDefaultNameFunc(),
			circuitry.ErrNameFnAlreadySet,
			"ErrNameFnAlreadySet",
		},
		"trip function conflict": {
			circuitry.WithDefaultTripFunc(),
			circuitry.ErrWillTripCircuitAlreadySet,
			"ErrWillTripCircuitAlreadySet",
		},
		"circuit specific error matcher conflict": {
			circuitry.WithCircuitSpecificErrorMatcher("circuit-b", circuitry.DefaultErrorMatcher),
			circuitry.ErrSettingConflict,
			"ErrSettingConflict",
		},
		"state change callback conflict": {
			circuitry.WithStateChangeCallback(func(name string, circuitContext map[string]any, from, to circuitry.CircuitState) {}),
			circuitry.ErrStateChangeCallbackAlreadySet,
			"ErrStorageBackendAlreadySet",
		},
		"failure count threshold": {
			circuitry.WithFailureCountThreshold(10),
			circuitry.ErrFailureCountThresholdAlreadySet,
			"ErrFailureCountThresholdAlreadySet",
		},
		"close threshold": {
			circuitry.WithCloseThreshold(5),
			circuitry.ErrCloseThresholdAlreadySet,
			"ErrCloseThresholdAlreadySet",
		},
		"logger": {
			circuitry.WithLogger(&log.NoOp{}),
			circuitry.ErrLoggerAlreadySet,
			"ErrLoggerAlreadySet",
		},
	}

	for name, testCase := range testCases {
		tc := testCase
		t.Run(name, func(t *testing.T) {
			_, err := circuitry.NewFactorySettings(tc.opt, tc.opt)
			if err == nil {
				t.Fatalf("expected %s trying to configure setting twice, but got nil", tc.expectedErrName)
			}
			if !errors.Is(err, tc.expectedErr) {
				t.Fatalf("expected to receive %s; got %v", tc.expectedErrName, err)
			}
		})
	}
}

func TestWithFailureCountThreshold(t *testing.T) {
	expected := uint64(10)
	s, err := circuitry.NewFactorySettings(circuitry.WithFailureCountThreshold(expected))
	if err != nil {
		t.Fatalf("expected to not receive an error but got %v", err)
	}
	if s.FailureCountThreshold != expected {
		t.Errorf("expected s.FailureCountThreshold == %d; got %d", expected, s.FailureCountThreshold)
	}
}

func TestWithAllowAfter(t *testing.T) {
	expected := time.Hour * 2
	s, err := circuitry.NewFactorySettings(circuitry.WithAllowAfter(expected))
	if err != nil {
		t.Fatalf("expected to not receive an error but got %v", err)
	}
	if s.AllowAfter != expected {
		t.Errorf("expected s.AllowAfter == %d; got %d", expected, s.AllowAfter)
	}
}

func TestWithCyclicClearAfter(t *testing.T) {
	expected := time.Hour * 24
	s, err := circuitry.NewFactorySettings(circuitry.WithCyclicClearAfter(expected))
	if err != nil {
		t.Fatalf("expected to not receive an error but got %v", err)
	}
	if s.CyclicClearAfter != expected {
		t.Errorf("expected s.CyclicClearAfter == %d; got %d", expected, s.CyclicClearAfter)
	}
}

func TestWithCircuitSpecificErrorMatcher(t *testing.T) {
	s, err := circuitry.NewFactorySettings(circuitry.WithCircuitSpecificErrorMatcher("circuit-a", circuitry.DefaultErrorMatcher), circuitry.WithCircuitSpecificErrorMatcher("circuit-b", func(err error) circuitry.ExecutionStatus {
		return circuitry.ExecutionSucceeded
	}), circuitry.WithDefaultFallbackErrorMatcher())
	if err != nil {
		t.Fatalf("expected to not receive an error but got %v", err)
	}
	if s.FallbackErrorMatcher == nil {
		t.Fatal("expected FallbackErrorMatcher == circuitry.DefaultErrorMatcher, but got nil")
	}
	matcher, ok := s.CircuitSpecificErrorMatcher["circuit-b"]
	if !ok {
		t.Fatal("expected a error matcher to be configured for \"circuit-b\" but didn't find one")
	}
	if matcher(fmt.Errorf("example %s", "test")) != circuitry.ExecutionSucceeded {
		t.Fatal("expected error matcher to always succeed but failed")
	}
}

func TestDefaultTripFunc(t *testing.T) {
	testCases := map[string]struct {
		threshold           uint64
		consecutiveFailures uint64
		willTrip            bool
	}{
		"does not trip on 0 failures":               {10, 0, false},
		"does not trip on equal number of failures": {10, 10, false},
		"trips on more failures than threshold":     {1, 2, true},
	}

	for name, testCase := range testCases {
		tc := testCase
		t.Run(name, func(t *testing.T) {
			ci := circuitry.CircuitInformation{ConsecutiveFailures: tc.consecutiveFailures}
			if actual := circuitry.DefaultTripFunc("c", tc.threshold, ci); actual != tc.willTrip {
				t.Errorf("expected DefaulTripFunc(_, %d, %+v) = %v; got %v", tc.threshold, ci, tc.willTrip, actual)
			}
		})
	}
}
