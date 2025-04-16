package circuitry

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/sigmavirus24/circuitry/log"
)

// WorkFn defines the allowed interface of a function that can be passed to
// [CircuitBreaker].Execute.
type WorkFn func() (any, error)

// CircuitBreaker defines the interface for an implementation of a circuit
// breaker.
type CircuitBreaker interface {
	// Start attempts to start work protected by the CircuitBreaker
	Start(context.Context) error
	// Name returns the name of the CircuitBreaker in use
	Name() string
	// Execute takes a function that does the work to be protected by the
	// CircuitBreaker and manages calling Start and End around the work
	// function for you. It returns the result of the function, the error if
	// returned by the function, and the error if returned from storing the
	// state in the backend. The third value may also be an error if the
	// CircuitBreaker is Open.
	Execute(context.Context, WorkFn) (workResult any, workErr error, circuitErr error)
	// End updates the status of the CircuitBreaker and returns an error if
	// there is an issue updating the storage backend
	End(context.Context, error) error
	// State returns the current state of the CircuitBreaker
	State(context.Context) (CircuitState, error)
	// Information returns the current CircuitInformation representing the
	// state of the CircuitBreaker
	Information(context.Context) (CircuitInformation, error)
}

type circuitBreaker struct {
	name                  string
	storage               StorageBackender
	circuitContext        map[string]any
	errMatcher            ExpectedErrorMatcherFunc
	failureCountThreshold uint64
	closeThreshold        uint64
	lock                  sync.Locker
	allowAfter            time.Duration
	resetCycle            time.Duration
	tripperFn             WillTripFunc
	stateChangeFn         StateChangeFunc
	logger                log.Logger

	counts     *circuitCounts
	state      CircuitState
	generation uint64
	expiry     time.Time
}

func (cb *circuitBreaker) Information(ctx context.Context) (CircuitInformation, error) {
	if err := cb.refreshFromRemoteState(ctx); err != nil {
		return CircuitInformation{}, err
	}
	return cb.toCircuitInformation(), nil
}

func (cb *circuitBreaker) Start(ctx context.Context) error {
	if cb.lock != nil {
		return ErrCircuitBreakerAlreadyStarted
	}
	if err := cb.lockRemoteState(ctx); err != nil {
		return err
	}
	if err := cb.refreshFromRemoteState(ctx); err != nil {
		return err
	}
	switch cb.state {
	case CircuitOpen:
		return ErrCircuitBreakerOpen
	case CircuitHalfOpen:
		if cb.counts.Total >= cb.closeThreshold {
			return ErrTooManyRequests
		}
	}
	cb.counts.AddRequest()
	cb.logger.WithField("circuit_name", cb.name).Info("starting circuit breaker")
	return nil
}

func (cb *circuitBreaker) refreshFromRemoteState(ctx context.Context) error {
	info, err := cb.storage.Retrieve(ctx, cb.name)
	if err != nil {
		return err
	}
	cb.fromCircuitInformation(info)
	now := time.Now()
	switch info.State {
	case CircuitClosed:
		if !info.ExpiresAfter.IsZero() && info.ExpiresAfter.Before(now) {
			cb.newGeneration(now)
		}
	case CircuitOpen:
		if info.ExpiresAfter.Before(now) {
			cb.setState(CircuitHalfOpen, now)
		}
	}
	return nil
}

func (cb *circuitBreaker) updateRemoteState(ctx context.Context, _ time.Time) error {
	info := cb.toCircuitInformation()
	err := cb.storage.Store(ctx, cb.name, info)
	if err != nil {
		return err
	}
	return nil
}

func (cb *circuitBreaker) lockRemoteState(ctx context.Context) error {
	lock, err := cb.storage.Lock(ctx, cb.name)
	if err != nil {
		return fmt.Errorf("cannot start circuit breaker for %s due to: %w", cb.name, err)
	}
	cb.lock = lock
	cb.lock.Lock()
	return nil
}

func (cb *circuitBreaker) unlockRemoteState() {
	cb.lock.Unlock()
	cb.lock = nil
}

func (cb *circuitBreaker) fromCircuitInformation(info CircuitInformation) {
	var zero time.Time
	cb.counts = fromCircuitInformation(info)
	cb.state = info.State
	cb.generation = info.Generation
	if info.ExpiresAfter.IsZero() && info.Generation == 0 && info.Total == 0 {
		if cb.resetCycle != 0 {
			cb.expiry = time.Now().Add(cb.resetCycle)
		} else {
			cb.expiry = zero
		}
	} else {
		cb.expiry = info.ExpiresAfter
	}
}

func (cb *circuitBreaker) toCircuitInformation() CircuitInformation {
	return cb.counts.ToCircuitInformation(cb.generation, cb.state, cb.expiry)
}

func (cb *circuitBreaker) Name() string {
	return cb.name
}

func (cb *circuitBreaker) End(ctx context.Context, err error) error {
	now := time.Now()
	defer cb.unlockRemoteState()
	status := cb.errMatcher(err)
	cb.logger.WithFields(log.Fields{
		"work_err":             err,
		"error_matcher_status": status.String(),
		"circuit_name":         cb.name,
	}).Info("circuit breaker ended")
	switch status {
	case ExecutionSucceeded:
		cb.endSuccess(ctx, now)
	default:
		cb.endFailure(ctx, now)
	}
	return cb.updateRemoteState(ctx, now)
}

func (cb *circuitBreaker) Execute(ctx context.Context, work WorkFn) (any, error, error) {
	err := cb.Start(ctx)
	if err != nil {
		return nil, nil, err
	}
	retVal, retErr := work()
	storageErr := cb.End(ctx, retErr)
	return retVal, retErr, storageErr
}

func (cb *circuitBreaker) endSuccess(_ context.Context, now time.Time) {
	switch cb.state {
	case CircuitClosed:
		cb.counts.AddSuccess()
	case CircuitHalfOpen:
		cb.counts.AddSuccess()
		if cb.counts.ConsecutiveSuccesses >= cb.closeThreshold {
			cb.setState(CircuitClosed, now)
		}
	}
}

func (cb *circuitBreaker) endFailure(_ context.Context, now time.Time) {
	switch cb.state {
	case CircuitClosed:
		cb.counts.AddFailure()
		if cb.tripperFn(cb.name, cb.failureCountThreshold, cb.toCircuitInformation()) {
			cb.setState(CircuitOpen, now)
		}
	case CircuitHalfOpen:
		cb.setState(CircuitOpen, now)
	}
}

func (cb *circuitBreaker) setState(state CircuitState, now time.Time) {
	if cb.state == state {
		return
	}
	prev := cb.state
	cb.state = state

	if (prev == CircuitHalfOpen && state != CircuitClosed) || state == CircuitHalfOpen {
		cb.updateExpiry(now)
	} else {
		cb.newGeneration(now)
	}
	if cb.stateChangeFn != nil {
		cb.stateChangeFn(cb.name, cb.circuitContext, prev, state)
	}
}

func (cb *circuitBreaker) updateExpiry(now time.Time) {
	var zero time.Time
	switch cb.state {
	case CircuitClosed:
		if cb.resetCycle == 0 {
			cb.expiry = zero
		} else {
			cb.expiry = now.Add(cb.resetCycle)
		}
	case CircuitOpen:
		cb.expiry = now.Add(cb.allowAfter)
	default:
		cb.expiry = zero
	}
}

func (cb *circuitBreaker) newGeneration(now time.Time) {
	cb.generation++
	cb.counts.Reset()

	cb.updateExpiry(now)
}

func (cb *circuitBreaker) State(ctx context.Context) (CircuitState, error) {
	if err := cb.refreshFromRemoteState(ctx); err != nil {
		return CircuitOpen, err
	}
	return cb.state, nil
}

var _ CircuitBreaker = (*circuitBreaker)(nil)

// CircuitBreakerFactory creates [CircuitBreaker]s for a given named circuit
type CircuitBreakerFactory struct {
	settings *FactorySettings
}

// NewCircuitBreakerFactory builds a new [CircuitBreakerFactory] from
// the [FactorySettings] supplied
func NewCircuitBreakerFactory(s *FactorySettings) *CircuitBreakerFactory {
	return &CircuitBreakerFactory{s}
}

// BreakerFor builds a new [CircuitBreaker] for the given named circuit and
// includes the circuit breaker context provided. The context is passed into
// the naming function and can be used by custom naming functions to produce
// names based off of a template
func (cbf *CircuitBreakerFactory) BreakerFor(name string, circuitContext map[string]any) CircuitBreaker {
	return cbf.settings.circuitBreakerFor(name, circuitContext)
}
