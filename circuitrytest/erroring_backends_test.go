package circuitrytest

import (
	"context"
	"errors"
	"testing"

	"github.com/sigmavirus24/circuitry"
)

func TestStoreError(t *testing.T) {
	storeErr := errors.New("cannot store value")
	testCases := map[string]struct {
		expectedErr error
	}{
		"nil":  {nil},
		"!nil": {storeErr},
	}

	for name, testCase := range testCases {
		tc := testCase
		t.Run(name, func(t *testing.T) {
			b := ErroringInMemoryBackend{StoreError: tc.expectedErr}
			if err := b.Store(context.TODO(), "", circuitry.CircuitInformation{}); err != tc.expectedErr {
				t.Fatalf("expected ErroringInMemoryBackend.Store() to return %v; got %v", tc.expectedErr, err)
			}
		})
	}
}

func TestRetrieveError(t *testing.T) {
	retrieveErr := errors.New("cannot retrieve value")
	testCases := map[string]struct {
		expectedErr error
	}{
		"nil":  {nil},
		"!nil": {retrieveErr},
	}

	for name, testCase := range testCases {
		tc := testCase
		t.Run(name, func(t *testing.T) {
			b := ErroringInMemoryBackend{RetrieveError: tc.expectedErr}
			if _, err := b.Retrieve(context.TODO(), ""); err != tc.expectedErr {
				t.Fatalf("expected ErroringInMemoryBackend.Retrieve() to return %v; got %v", tc.expectedErr, err)
			}
		})
	}
}

func TestLockError(t *testing.T) {
	lockErr := errors.New("cannot lock value")
	testCases := map[string]struct {
		expectedErr error
	}{
		"nil":  {nil},
		"!nil": {lockErr},
	}

	for name, testCase := range testCases {
		tc := testCase
		t.Run(name, func(t *testing.T) {
			b := ErroringInMemoryBackend{LockError: tc.expectedErr}
			lock, err := b.Lock(context.TODO(), "")
			if err != tc.expectedErr {
				t.Fatalf("expected ErroringInMemoryBackend.Lock() to return %v; got %v", tc.expectedErr, err)
			}
			if tc.expectedErr == nil && lock == nil {
				t.Fatalf("expected ErroringInMemoryBackend.Lock() to return non-nil Locker, got nil")
			}
		})
	}
}
