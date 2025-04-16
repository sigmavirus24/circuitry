package circuitry

import (
	"time"

	"github.com/sigmavirus24/circuitry/log"
)

// NameFunc defines the signature of the function used to generate names for
// circuits
type NameFunc func(circuit string, circuitContext map[string]any) string

// ExpectedErrorMatcherFunc defines the signature of a function that allows the
// user to define when an error is expected and the execution should not be
// considered to be failed
type ExpectedErrorMatcherFunc func(err error) ExecutionStatus

// WillTripFunc defines the signature of a function that allows the user to
// define when a [CircuitBreaker] may trip. It accepts the name of the
// [CircuitBreaker], the value of the FailureCountThreshold from the
// [FactorySettings], and the information about the state of the circuit. It
// returns `true` if the [CircuitBreaker] should transition to [CircuitOpen]
type WillTripFunc func(name string, configuredThreshold uint64, information CircuitInformation) bool

// StateChangeFunc defines the signature of a function that allows the user to
// define custom logic around the changing of [CircuitBreaker] state.
type StateChangeFunc func(name string, circuitContext map[string]any, from, to CircuitState)

// DefaultNameFunc is a default name implementation that simply uses the name of
// the circuit for the name of a given circuit breaker
func DefaultNameFunc(circuit string, circuitContext map[string]any) string {
	return circuit
}

// DefaultErrorMatcher is a default implementation of ExpectedErrorMatcher and
// assumes all errors are failed executions
func DefaultErrorMatcher(err error) ExecutionStatus {
	if _, ok := err.(*ExpectedConditionError); ok {
		return ExecutionSucceeded
	}
	if e, ok := err.(IsExpectedErrorer); ok {
		if e.IsExpected() {
			return ExecutionSucceeded
		}
	}
	switch err {
	case nil:
		return ExecutionSucceeded
	default:
		return ExecutionFailed
	}
}

// DefaultTripFunc is the default function used to determine if a
// [CircuitBreaker] is ready to trip. By default this returns true if the number
// of [CircuitInformation].ConsecutiveFailures is greater than the
// [FactorySettings].FailureCountThreshold.
func DefaultTripFunc(_ string, configuredThreshold uint64, information CircuitInformation) bool {
	return information.ConsecutiveFailures > configuredThreshold
}

// FactorySettings contains information for configuring a CircuitBreakerFactory and
// any CircuitBreaker it creates.
type FactorySettings struct {
	StorageBackend              StorageBackender                    // StorageBackend allows for remote storage of [CircuitBreaker] state.
	NameFn                      NameFunc                            // NameFn allows the [CircuitBreakerFactory] to dynamically generate names.
	FallbackErrorMatcher        ExpectedErrorMatcherFunc            // FallbackErrorMatcher which provides for a default error matcher if one isn't found for a specific [CircuitBreaker] by name.
	CircuitSpecificErrorMatcher map[string]ExpectedErrorMatcherFunc // CircuitSpecificErrorMatcher provides a way for different [CircuitBreaker]s to have specific error matchers.
	FailureCountThreshold       uint64                              // FailureCountThreshold defines the threshold after which a [CircuitBreaker] transitions to the [CircuitOpen] [CircuitState].
	CloseThreshold              uint64                              // CloseThreshold defines the number of successful requests after which a [CircuitBreaker] in the [CircuitHalfOpen] [CircuitState] returns to [CircuitClosed] [CircuitState].
	AllowAfter                  time.Duration                       // AllowAfter defines the time after which a [CircuitBreaker] in the [CircuitOpen] [CircuitState] transitions to [CircuitHalfOpen]. If not specified, the default is 60 seconds.
	CyclicClearAfter            time.Duration                       // CyclicClearAfter defines the time after which a [CircuitBreaker] resets its internal counts. If not specified, the [CircuitBreaker] never resets its internal counts.
	StateChangeCallback         StateChangeFunc                     // StateChangeCallback stores a callback for users to learn when the [CircuitBreaker] state is changing.
	WillTripCircuit             WillTripFunc                        // WillTripCircuit provides a way to customize whether the [WillTripCircuit] will trip in conjunction with the [FailureCountThreshold].
	Logger                      log.Logger                          // Logger allows the caller to specify a given logger to use for all [CircuitBreaker]s.
}

// GenerateName builds a name for a [CircuitBreaker]
func (s *FactorySettings) GenerateName(circuit string, circuitContext map[string]any) string {
	if s.NameFn == nil {
		return DefaultNameFunc(circuit, circuitContext)
	}
	return s.NameFn(circuit, circuitContext)
}

// circuitBreakerFor builds a [CircuitBreaker] from the settings configured
// globally
func (s *FactorySettings) circuitBreakerFor(circuit string, circuitContext map[string]any) CircuitBreaker {
	name := s.GenerateName(circuit, circuitContext)
	matcher, ok := s.CircuitSpecificErrorMatcher[circuit]
	if !ok {
		if s.FallbackErrorMatcher != nil {
			matcher = s.FallbackErrorMatcher
		} else {
			matcher = DefaultErrorMatcher
		}
	}
	tripper := s.WillTripCircuit
	if tripper == nil {
		tripper = DefaultTripFunc
	}
	logger := s.Logger
	if logger == nil {
		logger = &log.NoOp{}
	}
	return &circuitBreaker{
		name:                  name,
		storage:               s.StorageBackend,
		errMatcher:            matcher,
		failureCountThreshold: s.FailureCountThreshold,
		closeThreshold:        s.CloseThreshold,
		allowAfter:            s.AllowAfter,
		resetCycle:            s.CyclicClearAfter,
		lock:                  nil,
		circuitContext:        circuitContext,
		tripperFn:             tripper,
		stateChangeFn:         s.StateChangeCallback,
		logger:                logger,
	}
}

// NewFactorySettings constructs a [FactorySettings] struct with the provided options
func NewFactorySettings(opts ...SettingsOption) (*FactorySettings, error) {
	s := &FactorySettings{CircuitSpecificErrorMatcher: make(map[string]ExpectedErrorMatcherFunc), AllowAfter: 0 * time.Second}
	for _, opt := range opts {
		err := opt(s)
		if err != nil {
			return nil, err
		}
	}
	return s, nil
}

// SettingsOption configures an option on a setting
type SettingsOption func(s *FactorySettings) error

// WithDefaultFallbackErrorMatcher configures the FallbackErrorMatcher setting a the default
// value, [DefaultErrorMatcher]
func WithDefaultFallbackErrorMatcher() SettingsOption {
	return WithFallbackErrorMatcher(DefaultErrorMatcher)
}

// WithFallbackErrorMatcher configures the FallbackErrorMatcher with the
// provided matcher.
func WithFallbackErrorMatcher(matcher ExpectedErrorMatcherFunc) SettingsOption {
	return func(s *FactorySettings) error {
		if s.FallbackErrorMatcher != nil {
			return ErrFallbackErrorMatcherAlreadySet
		}
		s.FallbackErrorMatcher = matcher
		return nil
	}
}

// WithCircuitSpecificErrorMatcher sets a specific error matcher for a given
// circuit with the value provided.
func WithCircuitSpecificErrorMatcher(circuitName string, matcher ExpectedErrorMatcherFunc) SettingsOption {
	return func(s *FactorySettings) error {
		if _, ok := s.CircuitSpecificErrorMatcher[circuitName]; ok {
			return newCircuitSpecificSettingsConflictError("CircuitSpecificErrorMatcher", circuitName)
		}
		s.CircuitSpecificErrorMatcher[circuitName] = matcher
		return nil
	}
}

// WithDefaultNameFunc configure the NameFn setting with a default value,
// [DefaultNameFunc].
func WithDefaultNameFunc() SettingsOption {
	return WithNameFunc(DefaultNameFunc)
}

// WithNameFunc configures the NameFn setting with the provided [NameFunc].
func WithNameFunc(namefn NameFunc) SettingsOption {
	return func(s *FactorySettings) error {
		if s.NameFn != nil {
			return ErrNameFnAlreadySet
		}
		s.NameFn = namefn
		return nil
	}
}

// WithFailureCountThreshold configures the threshold where the circuit
// breaker trips to [CircuitOpen].
func WithFailureCountThreshold(threshold uint64) SettingsOption {
	return func(s *FactorySettings) error {
		if s.FailureCountThreshold > 0 {
			return ErrFailureCountThresholdAlreadySet
		}
		s.FailureCountThreshold = threshold
		return nil
	}
}

// WithCloseThreshold configures the number of consecutive successful requests
// needed for a [CircuitHalfOpen] [CircuitBreaker] to transition to
// [CircuitClosed].
func WithCloseThreshold(threshold uint64) SettingsOption {
	return func(s *FactorySettings) error {
		if s.CloseThreshold > 0 {
			return ErrCloseThresholdAlreadySet
		}
		s.CloseThreshold = threshold
		return nil
	}
}

// WithAllowAfter configures the AllowAfter setting and always overrides it.
func WithAllowAfter(duration time.Duration) SettingsOption {
	return func(s *FactorySettings) error {
		s.AllowAfter = duration
		return nil
	}
}

// WithCyclicClearAfter configures the CyclicClearAfter setting and always overrides it.
func WithCyclicClearAfter(duration time.Duration) SettingsOption {
	return func(s *FactorySettings) error {
		s.CyclicClearAfter = duration
		return nil
	}
}

// WithStateChangeCallback configures the StateChangeCallback setting with the
// provided [StateChangeFunc] if it's not already set
func WithStateChangeCallback(cb StateChangeFunc) SettingsOption {
	return func(s *FactorySettings) error {
		if s.StateChangeCallback != nil {
			return ErrStateChangeCallbackAlreadySet
		}
		s.StateChangeCallback = cb
		return nil
	}
}

// WithDefaultTripFunc sets the WillTripCircuit setting to the default
// function [DefaultTripFunc].
func WithDefaultTripFunc() SettingsOption {
	return WithTripFunc(DefaultTripFunc)
}

// WithTripFunc configures the WillTripCircuit setting to have custom logic
// for when a [CircuitBreaker] trips open
func WithTripFunc(tf WillTripFunc) SettingsOption {
	return func(s *FactorySettings) error {
		if s.WillTripCircuit != nil {
			return ErrWillTripCircuitAlreadySet
		}
		s.WillTripCircuit = tf
		return nil
	}
}

// WithLogger configures the Logger setting to use a given logger as long as
// it implements the interface we expect
func WithLogger(l log.Logger) SettingsOption {
	return func(s *FactorySettings) error {
		if s.Logger != nil {
			return ErrLoggerAlreadySet
		}
		s.Logger = l
		return nil
	}
}
