package circuitry

import (
	"fmt"
)

type constError string

func (c constError) Error() string {
	return string(c)
}

var _ error = (*constError)(nil)

const (
	// ErrSettingConflict is returned when a setting has already been set and
	// the function refuses to override it
	ErrSettingConflict = constError("setting has already been configured via an option function, refusing to override")
	// ErrCircuitBreakerAlreadyStarted is returned when a CircuitBreaker has
	// already been started
	ErrCircuitBreakerAlreadyStarted = constError("circuit breaker has already been started and is executing")
	// ErrCircuitBreakerOpen is returned when a CircuitBreaker has already
	// been tripped and is in the open state
	ErrCircuitBreakerOpen = constError("circuit breaker is open")
	// ErrTooManyRequests is returned when a CircuitBreaker is in the
	// CircuitHalfOpen state and too many requests have been made
	ErrTooManyRequests = constError("too many requests with a circuit breaker in the half-open state")
	// ErrProvisioningStorageBackend is returned when a StorageBackend
	// encounters an issue during it's creation that is not a setting conflict
	ErrProvisioningStorageBackend = constError("could not provision storage backend")
)

// SettingsConflictError contains the FactorySettingsName in the error and
// provides an overarching type for all errors resulting from a conflict
type SettingsConflictError struct {
	FactorySettingsName string
	formatStr           string
}

const sceFormatStr = "%s %s"

func newSettingsConflictError(name string) SettingsConflictError {
	return SettingsConflictError{
		FactorySettingsName: name,
		formatStr:           sceFormatStr,
	}
}

func (sce SettingsConflictError) Error() string {
	formatStr := sce.formatStr
	if formatStr == "" {
		formatStr = sceFormatStr
	}
	return fmt.Sprintf(formatStr, sce.FactorySettingsName, ErrSettingConflict)
}

func (sce SettingsConflictError) Unwrap() error {
	return ErrSettingConflict
}

// CircuitSpecificSettingsConflictError represents a conflict for a
// FactorySettings for something that is specific to a given CircuitBreaker
type CircuitSpecificSettingsConflictError struct {
	SettingsConflictError
	CircuitName string
}

const cssceFormatStr = "%s already registered for %s: %s"

func newCircuitSpecificSettingsConflictError(settingsName string, circuitName string) CircuitSpecificSettingsConflictError {
	return CircuitSpecificSettingsConflictError{
		SettingsConflictError: SettingsConflictError{
			FactorySettingsName: settingsName,
			formatStr:           cssceFormatStr,
		},
		CircuitName: circuitName,
	}
}

func (e CircuitSpecificSettingsConflictError) Error() string {
	formatStr := e.formatStr
	if formatStr == "" {
		formatStr = cssceFormatStr
	}
	return fmt.Sprintf(formatStr, e.FactorySettingsName, e.CircuitName, ErrSettingConflict)
}

func (e CircuitSpecificSettingsConflictError) Unwrap() error {
	return e.SettingsConflictError.Unwrap()
}

var _ error = (*SettingsConflictError)(nil)
var _ error = (*CircuitSpecificSettingsConflictError)(nil)

var (
	// ErrStorageBackendAlreadySet is returned when the StorageBackend setting
	// has already been configured
	ErrStorageBackendAlreadySet = newSettingsConflictError("StorageBackend")
	// ErrFallbackErrorMatcherAlreadySet is returned when the ErrorMatcher setting has
	// already been configured
	ErrFallbackErrorMatcherAlreadySet = newSettingsConflictError("FallbackErrorMatcher")
	// ErrNameFnAlreadySet is returned when the NameFn setting has already been
	// configured
	ErrNameFnAlreadySet = newSettingsConflictError("NameFn")
	// ErrStateChangeCallbackAlreadySet is returned when the
	// StateChangeCallback setting has already been configured
	ErrStateChangeCallbackAlreadySet = newSettingsConflictError("StateChangeCallback")
	// ErrFailureCountThresholdAlreadySet is returned when the
	// FailureCountThreshold setting has already been configured
	ErrFailureCountThresholdAlreadySet = newSettingsConflictError("FailureCountThreshold")
	// ErrCloseThresholdAlreadySet is returned when the
	// CloseThreshold setting has already been configured
	ErrCloseThresholdAlreadySet = newSettingsConflictError("CloseThreshold")
	// ErrWillTripCircuitAlreadySet is returned when the WillTripCircuit
	// setting has already been configured
	ErrWillTripCircuitAlreadySet = newSettingsConflictError("WillTripCircuit")
	// ErrLoggerAlreadySet is returned when the Logger
	// setting has already been configured
	ErrLoggerAlreadySet = newSettingsConflictError("Logger")
)

// IsExpectedErrorer defines an interface that one can use when defining their
// own concrete error types. It allows users to add an IsExpected() method to
// their function to signal to cicuitry that the error is one that should not
// count as a failure.
type IsExpectedErrorer interface {
	IsExpected() bool
}

// ExpectedConditionError is provided so users can intentionally wrap an error
// they do not want to count as a failure without having to create additional
// error matchers
type ExpectedConditionError struct {
	err error
}

// WrapExpectedConditionError wraps an error with the ExpectedConditionError
// type to signal that the error should not cause a failure.
func WrapExpectedConditionError(err error) error {
	return &ExpectedConditionError{err}
}

func (e *ExpectedConditionError) Error() string {
	return e.err.Error()
}

func (e *ExpectedConditionError) Unwrap() error {
	return e.err
}

// IsExpected indicates that this is an expected error condition
func (e *ExpectedConditionError) IsExpected() bool {
	return true
}

var _ error = (*ExpectedConditionError)(nil)
var _ IsExpectedErrorer = (*ExpectedConditionError)(nil)
