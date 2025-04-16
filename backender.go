package circuitry

import (
	"context"
	"sync"
)

// StorageBackender defines the contract expected of a Storage Backend for the
// [CircuitBreaker] to use to store information.
type StorageBackender interface {
	// Store the circuit information for the given string name
	Store(context.Context, string, CircuitInformation) error
	// Retrieve the circuit information
	Retrieve(context.Context, string) (CircuitInformation, error)
	// Lock provides a lock for the given name to allow for atomicity via the
	// backend
	Lock(context.Context, string) (sync.Locker, error)
}

// WithStorageBackend allows the user to specify a [StorageBackender]
// implementation for [CircuitBreaker]s.
func WithStorageBackend(backend StorageBackender) SettingsOption {
	return func(s *FactorySettings) error {
		if s.StorageBackend != nil {
			return ErrStorageBackendAlreadySet
		}
		s.StorageBackend = backend
		return nil
	}
}
