package circuitry_test

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/sigmavirus24/circuitry"
	"github.com/sigmavirus24/circuitry/backends"
	"github.com/sigmavirus24/circuitry/circuitrytest"
)

func newFactory(opts ...circuitry.SettingsOption) *circuitry.CircuitBreakerFactory {
	settings, _ := circuitry.NewFactorySettings(opts...)
	return circuitry.NewCircuitBreakerFactory(settings)
}

func TestCircuitBreakerFactoryBasic(t *testing.T) {
	const expectedName = "example"
	factory := newFactory(backends.WithInMemoryBackend())
	breaker := factory.BreakerFor(expectedName, map[string]any{})
	if breaker == nil {
		t.Fatal("expected a CircuitBreaker but got nil")
	}
	if name := breaker.Name(); name != expectedName {
		t.Fatalf("expected CircuitBreaker.Name() = %s; got %s", expectedName, name)
	}
	if state, _ := breaker.State(context.TODO()); state != circuitry.CircuitClosed {
		t.Fatalf("expected CircuitBreaker.State() = %s; got %s", circuitry.CircuitClosed, state)
	}
}

func TestCircuitBreakerFactoryOptions(t *testing.T) {
	testCases := map[string]struct {
		options        []circuitry.SettingsOption
		circuitName    string
		circuitContext map[string]any
		expectedName   string
		expectedState  circuitry.CircuitState
	}{
		"custom namefn": {
			[]circuitry.SettingsOption{
				backends.WithInMemoryBackend(),
				circuitry.WithNameFunc(func(name string, circuitContext map[string]any) string {
					tenantID, ok := circuitContext["tenant"].(string)
					if !ok {
						return name
					}
					return fmt.Sprintf("%s::%s", name, tenantID)
				}),
			},
			"example",
			map[string]any{"tenant": "20b60248-ec14-4c2c-a86f-ad3fbd6fac54"},
			"example::20b60248-ec14-4c2c-a86f-ad3fbd6fac54",
			circuitry.CircuitClosed,
		},
		"custom matcher": {
			[]circuitry.SettingsOption{
				backends.WithInMemoryBackend(),
				circuitry.WithCircuitSpecificErrorMatcher("example", func(err error) circuitry.ExecutionStatus { return circuitry.ExecutionFailed }),
				circuitry.WithNameFunc(func(name string, circuitContext map[string]any) string {
					tenantID, ok := circuitContext["tenant"].(string)
					if !ok {
						return name
					}
					return fmt.Sprintf("%s::%s", name, tenantID)
				}),
			},
			"example",
			map[string]any{"tenant": "20b60248-ec14-4c2c-a86f-ad3fbd6fac54"},
			"example::20b60248-ec14-4c2c-a86f-ad3fbd6fac54",
			circuitry.CircuitClosed,
		},
		"fallback matcher": {
			[]circuitry.SettingsOption{
				backends.WithInMemoryBackend(),
				circuitry.WithFallbackErrorMatcher(circuitry.DefaultErrorMatcher),
				circuitry.WithCircuitSpecificErrorMatcher("example2", func(err error) circuitry.ExecutionStatus { return circuitry.ExecutionFailed }),
				circuitry.WithNameFunc(func(name string, circuitContext map[string]any) string {
					tenantID, ok := circuitContext["tenant"].(string)
					if !ok {
						return name
					}
					return fmt.Sprintf("%s::%s", name, tenantID)
				}),
			},
			"example",
			map[string]any{"tenant": "20b60248-ec14-4c2c-a86f-ad3fbd6fac54"},
			"example::20b60248-ec14-4c2c-a86f-ad3fbd6fac54",
			circuitry.CircuitClosed,
		},
	}

	for name, testCase := range testCases {
		tc := testCase
		t.Run(name, func(t *testing.T) {
			factory := newFactory(tc.options...)
			breaker := factory.BreakerFor(tc.circuitName, tc.circuitContext)
			if breaker == nil {
				t.Fatal("expected a CircuitBreaker but got nil")
			}
			if name := breaker.Name(); name != tc.expectedName {
				t.Fatalf("expected CircuitBreaker.Name() = %s; got %s", tc.expectedName, name)
			}
			if state, _ := breaker.State(context.TODO()); state != circuitry.CircuitClosed {
				t.Fatalf("expected CircuitBreaker.State() = %s; got %s", circuitry.CircuitClosed, state)
			}
		})
	}
}

func TestCircuitBreakerStateErroringBackend(t *testing.T) {
	factory := newFactory(circuitry.WithStorageBackend(circuitrytest.ErroringInMemoryBackend{RetrieveError: errors.New("cannot retrieve from backend")}))
	breaker := factory.BreakerFor("name", map[string]any{})
	if state, err := breaker.State(context.TODO()); state != circuitry.CircuitOpen && err == nil {
		t.Fatalf("expected breaker.State() to fail closed and return error from backend; got state = %s && err = %v", state, err)
	}
	if _, err := breaker.Information(context.TODO()); err == nil {
		t.Fatalf("expected breaker.Information() to fail with an error; got err = %v", err)
	}
}

func TestCircuitBreakerStateFailedStart(t *testing.T) {
	testCases := map[string]struct {
		lockErr error
		retrErr error
	}{
		"fail to lock":     {errors.New("cannot retrieve lock form backend"), nil},
		"fail to retrieve": {nil, errors.New("cannot refresh state from backend")},
	}
	for name, testCase := range testCases {
		tc := testCase
		t.Run(name, func(t *testing.T) {
			factory := newFactory(
				circuitry.WithStorageBackend(
					circuitrytest.ErroringInMemoryBackend{
						LockError:     tc.lockErr,
						RetrieveError: tc.retrErr,
					}))
			breaker := factory.BreakerFor("name", map[string]any{})
			if err := breaker.Start(context.TODO()); err == nil {
				t.Fatal("expected breaker.Start() to fail and return error from backend; got err = nil")
			}
		})
	}
}

func TestCircuitBreakerStartFailsAfterStarted(t *testing.T) {
	factory := newFactory(backends.WithInMemoryBackend())
	breaker := factory.BreakerFor("name", map[string]any{})
	if err := breaker.Start(context.TODO()); err != nil {
		t.Fatalf("expected breaker.Start() to succeed; got err = %v", err)
	}
	if err := breaker.Start(context.TODO()); err == nil {
		t.Fatalf("expected second call to breaker.Start() to fail, got err = nil")
	}
}

func TestCircuitBreakerExecute(t *testing.T) {
	testCases := map[string]struct {
		workFn     circuitry.WorkFn
		workFailed bool
	}{
		"success": {func() (any, error) { return struct{}{}, nil }, false},
		"failure": {func() (any, error) { return nil, errors.New("failed to do work") }, true},
	}
	for name, testCase := range testCases {
		tc := testCase
		t.Run(name, func(t *testing.T) {
			factory := newFactory(backends.WithInMemoryBackend())
			breaker := factory.BreakerFor("name", map[string]any{})
			_, retErr, breakerErr := breaker.Execute(context.TODO(), tc.workFn)
			if breakerErr != nil {
				t.Fatalf("expected breaker.Execute() to succeed; got breakerErr = %v", breakerErr)
			}
			if retErr != nil && !tc.workFailed {
				t.Fatalf("expected breaker.Execute(ctx, WorkFn) to return nil WorkFn err; got retErr = %v", retErr)
			}
			if retErr == nil && tc.workFailed {
				t.Fatal("expected breaker.Execute(ctx, WorkFn) to return non-nil WorkFn err; got retErr = nil")
			}
		})
	}
}

func TestCircuitBreakerExecuteFails(t *testing.T) {
	testCases := map[string]struct {
		backend circuitry.StorageBackender
	}{
		"can't lock":     {circuitrytest.ErroringInMemoryBackend{LockError: errors.New("cannot retrieve lock from backend")}},
		"can't retrieve": {circuitrytest.ErroringInMemoryBackend{RetrieveError: errors.New("cannot refresh state from backend")}},
		"can't store":    {circuitrytest.ErroringInMemoryBackend{StoreError: errors.New("cannot store state in backend")}},
	}

	for name, testCase := range testCases {
		tc := testCase
		t.Run(name, func(t *testing.T) {
			factory := newFactory(circuitry.WithStorageBackend(tc.backend))
			breaker := factory.BreakerFor("name", map[string]any{})
			_, _, breakerErr := breaker.Execute(context.TODO(), func() (any, error) {
				return struct{}{}, nil
			})
			if breakerErr == nil {
				t.Fatal("expected to fail on storage internals; but got breakerErr = nil")
			}
		})
	}

}

func TestSuccessCountTracking(t *testing.T) {
	factory := newFactory(backends.WithInMemoryBackend(), circuitry.WithFailureCountThreshold(10), circuitry.WithCloseThreshold(5))
	breaker := factory.BreakerFor("name", map[string]any{})

	for i := 0; i < 20; i++ {
		if err := breaker.Start(context.TODO()); err != nil {
			t.Fatalf("couldn't start circuit breaker but should have been able to; got %v", err)
		}
		if err := breaker.End(context.TODO(), nil); err != nil {
			t.Fatalf("couldn't end circuit breaker but should have been able to; got %v", err)
		}
		if state, err := breaker.State(context.TODO()); err != nil || state != circuitry.CircuitClosed {
			t.Fatalf("breaker state should be closed; got state = %s, err = %v", state, err)
		}
		expected := uint64(i + 1)
		if ci, _ := breaker.Information(context.TODO()); ci.Total != expected || ci.ConsecutiveSuccesses != expected || ci.TotalSuccesses != expected {
			t.Fatalf("breaker counts incorrect, expected %d; got ci.Total = %d, ci.ConsecutiveSuccesses = %d, ci.TotalSuccesses = %d", expected, ci.Total, ci.ConsecutiveSuccesses, ci.TotalSuccesses)
		}
	}
}

func TestBreakerOpening(t *testing.T) {
	var tripped bool
	var stateChanged bool
	willTripFn := func(name string, threshold uint64, info circuitry.CircuitInformation) bool {
		if info.ConsecutiveFailures >= threshold {
			tripped = true
			return true
		}
		return false
	}
	stateChangeFn := func(name string, circuitContext map[string]any, from, to circuitry.CircuitState) {
		stateChanged = true
	}
	factory := newFactory(
		backends.WithInMemoryBackend(),
		circuitry.WithFailureCountThreshold(10),
		circuitry.WithCloseThreshold(5),
		circuitry.WithTripFunc(willTripFn),
		circuitry.WithStateChangeCallback(stateChangeFn),
		circuitry.WithAllowAfter(5*time.Minute),
	)
	breaker := factory.BreakerFor("name", map[string]any{})

	for i := 0; i < 9; i++ {
		if err := breaker.Start(context.TODO()); err != nil {
			t.Fatalf("couldn't start circuit breaker but should have been able to; got %v", err)
		}
		if err := breaker.End(context.TODO(), errors.New("test")); err != nil {
			t.Fatalf("couldn't end circuit breaker but should have been able to; got %v", err)
		}
		if state, err := breaker.State(context.TODO()); err != nil || state != circuitry.CircuitClosed {
			t.Fatalf("breaker state should be closed; got state = %s, err = %v", state, err)
		}
		expected := uint64(i + 1)
		if ci, _ := breaker.Information(context.TODO()); ci.Total != expected || ci.ConsecutiveFailures != expected || ci.TotalFailures != expected {
			t.Fatalf("breaker counts incorrect, expected %d; got ci.Total = %d, ci.ConsecutiveFailures = %d, ci.TotalFailures = %d", expected, ci.Total, ci.ConsecutiveFailures, ci.TotalFailures)
		}
	}
	if err := breaker.Start(context.TODO()); err != nil {
		t.Fatalf("couldn't start circuit breaker but should have been able to; got %v", err)
	}
	if err := breaker.End(context.TODO(), errors.New("test")); err != nil {
		t.Fatalf("couldn't end circuit breaker but should have been able to; got %v", err)
	}
	if state, err := breaker.State(context.TODO()); err != nil || state != circuitry.CircuitOpen {
		t.Fatalf("breaker state should be open; got state = %s, err = %v", state, err)
	}
	if !tripped {
		t.Fatal("expected willTripFn to have been called and circuit to have tripped, but it didn't")
	}
	if !stateChanged {
		t.Fatal("expected stateChangeFn to have been called, but it wasn't")
	}
	ci, err := breaker.Information(context.TODO())
	if err != nil {
		t.Fatalf("expected to get information; got err = %v", err)
	}
	if ci.Generation < 1 {
		t.Fatalf("expected breaker generation to be greater or equal to 1; got %d", ci.Generation)
	}

	if err := breaker.Start(context.TODO()); !errors.Is(err, circuitry.ErrCircuitBreakerOpen) {
		t.Fatalf("expected breaker.Start() = ErrCircuitBreakerOpen(%s); got %v", circuitry.ErrCircuitBreakerOpen, err)
	}
}

func TestHalfOpenToClosed(t *testing.T) {
	factory := newFactory(
		backends.WithInMemoryBackend(),
		circuitry.WithFailureCountThreshold(1),
		circuitry.WithCloseThreshold(2),
		circuitry.WithDefaultTripFunc(),
		circuitry.WithAllowAfter(5*time.Millisecond),
	)
	breaker := factory.BreakerFor("TestHalfOpenToClosed", map[string]any{})
	alwaysErrorFn := func() (any, error) { return nil, errors.New("test") }
	neverErrorFn := func() (any, error) { return nil, nil }

	for i := 0; i < 2; i++ {
		if _, _, err := breaker.Execute(context.TODO(), alwaysErrorFn); err != nil {
			t.Fatalf("couldn't execute work function; got %v", err)
		}
	}
	if state, err := breaker.State(context.TODO()); err != nil || state != circuitry.CircuitOpen {
		t.Fatalf("breaker state should be open; got state = %s, err = %v", state, err)
	}
	time.Sleep(300 * time.Millisecond)
	ci, err := breaker.Information(context.TODO())
	if err != nil {
		t.Fatalf("expected information got err = %v", err)
	}
	if ci.Generation < 1 {
		t.Fatalf("expected breaker generation to be greater or equal to 1; got %d", ci.Generation)
	}
	if ci.State != circuitry.CircuitHalfOpen {
		t.Fatalf("expected breaker to be in half-open state; got ci.State = %s", ci.State)
	}
	for i := 0; i < 2; i++ {
		if _, _, err := breaker.Execute(context.TODO(), neverErrorFn); err != nil {
			t.Fatalf("couldn't execute work function; got %v", err)
		}
	}
	ci, err = breaker.Information(context.TODO())
	if err != nil {
		t.Fatalf("expected information got err = %v", err)
	}
	if ci.State != circuitry.CircuitClosed {
		t.Fatalf("expected ci.State = %s; got %s", circuitry.CircuitClosed, ci.State)
	}
}

func TestHalfOpenToOpen(t *testing.T) {
	factory := newFactory(
		backends.WithInMemoryBackend(),
		circuitry.WithFailureCountThreshold(1),
		circuitry.WithCloseThreshold(2),
		circuitry.WithDefaultTripFunc(),
		circuitry.WithAllowAfter(5*time.Millisecond),
	)
	breaker := factory.BreakerFor("TestHalfOpenToOpen", map[string]any{})
	alwaysErrorFn := func() (any, error) { return nil, errors.New("test") }
	neverErrorFn := func() (any, error) { return nil, nil }

	for i := 0; i < 2; i++ {
		if _, _, err := breaker.Execute(context.TODO(), alwaysErrorFn); err != nil {
			t.Fatalf("couldn't execute work function; got %v", err)
		}
	}
	if state, err := breaker.State(context.TODO()); err != nil || state != circuitry.CircuitOpen {
		t.Fatalf("breaker state should be open; got state = %s, err = %v", state, err)
	}
	time.Sleep(300 * time.Millisecond)
	ci, err := breaker.Information(context.TODO())
	if err != nil {
		t.Fatalf("expected information got err = %v", err)
	}
	if ci.Generation < 1 {
		t.Fatalf("expected breaker generation to be greater or equal to 1; got %d", ci.Generation)
	}
	if ci.State != circuitry.CircuitHalfOpen {
		t.Fatalf("expected breaker to be in half-open state; got ci.State = %s", ci.State)
	}
	if _, _, err := breaker.Execute(context.TODO(), neverErrorFn); err != nil {
		t.Fatalf("couldn't execute work function; got %v", err)
	}
	if _, _, err := breaker.Execute(context.TODO(), alwaysErrorFn); err != nil {
		t.Fatalf("couldn't execute work function; got %v", err)
	}
	ciOpen, err := breaker.Information(context.TODO())
	if err != nil {
		t.Fatalf("expected information got err = %v", err)
	}
	if ciOpen.State != circuitry.CircuitOpen {
		t.Fatalf("expected ciOpen.State = %s; got %s", circuitry.CircuitOpen, ciOpen.State)
	}
	if ciOpen.Generation != ci.Generation {
		t.Fatalf("expected generation not to change transitioning back to CircuitOpen; expected Generation = %d, got %d", ci.Generation, ciOpen.Generation)
	}
}

func TestCyclicClearing(t *testing.T) {
	factory := newFactory(
		backends.WithInMemoryBackend(),
		circuitry.WithFailureCountThreshold(1),
		circuitry.WithCloseThreshold(2),
		circuitry.WithDefaultTripFunc(),
		circuitry.WithCyclicClearAfter(5*time.Millisecond),
	)
	breaker := factory.BreakerFor("TestCyclicClearing", map[string]any{})
	neverErrorFn := func() (any, error) { return nil, nil }

	for i := 0; i < 2; i++ {
		if _, _, err := breaker.Execute(context.TODO(), neverErrorFn); err != nil {
			t.Fatalf("couldn't execute work function; got %v", err)
		}
	}
	time.Sleep(100 * time.Millisecond)
	ci, err := breaker.Information(context.TODO())
	if err != nil {
		t.Fatalf("expected information but got err = %v", err)
	}
	if ci.Generation != 1 {
		t.Fatalf("expected new generation after cyclic clearing, got original; ci = %+v", ci)
	}
}

func TestTooManyRequestsInHalfOpen(t *testing.T) {
	factory := newFactory(
		backends.WithInMemoryBackend(),
		circuitry.WithFailureCountThreshold(1),
		circuitry.WithCloseThreshold(2),
		circuitry.WithDefaultTripFunc(),
		circuitry.WithCyclicClearAfter(50*time.Millisecond),
		circuitry.WithAllowAfter(50*time.Millisecond),
	)
	breaker := factory.BreakerFor("TestTooManyRequestsInHalfOpen", map[string]any{})
	alwaysErrorFn := func() (any, error) { return nil, errors.New("test") }
	neverErrorFn := func() (any, error) { return nil, nil }

	for i := 0; i < 2; i++ {
		if _, _, err := breaker.Execute(context.TODO(), alwaysErrorFn); err != nil {
			t.Fatalf("couldn't execute work function; got %v", err)
		}
	}
	if state, err := breaker.State(context.TODO()); err != nil || state != circuitry.CircuitOpen {
		t.Fatalf("breaker state should be open; got state = %s, err = %v", state, err)
	}
	time.Sleep(100 * time.Millisecond)
	ci, err := breaker.Information(context.TODO())
	if err != nil {
		t.Fatalf("expected information but got err = %v", err)
	}
	if ci.State != circuitry.CircuitHalfOpen {
		t.Fatalf("expected breaker to be in half-open state; got ci.State = %s", ci.State)
	}
	// Succeed and then go back to open
	if _, _, err := breaker.Execute(context.TODO(), neverErrorFn); err != nil {
		t.Fatalf("couldn't execute work function; got %v", err)
	}
	if _, _, err := breaker.Execute(context.TODO(), alwaysErrorFn); err != nil {
		t.Fatalf("couldn't execute work function; got %v", err)
	}
	if state, err := breaker.State(context.TODO()); err != nil || state != circuitry.CircuitOpen {
		t.Fatalf("breaker state should be open; got state = %s, err = %v", state, err)
	}
	time.Sleep(100 * time.Millisecond)
	ci, err = breaker.Information(context.TODO())
	if err != nil {
		t.Fatalf("expected information got err = %v", err)
	}
	if ci.State != circuitry.CircuitHalfOpen {
		t.Fatalf("expected breaker to be in half-open state; got ci.State = %s", ci.State)
	}
	if _, _, err := breaker.Execute(context.TODO(), alwaysErrorFn); !errors.Is(err, circuitry.ErrTooManyRequests) {
		t.Fatalf("expected breaker to error on too many requests; got err = %v", err)
	}
}
