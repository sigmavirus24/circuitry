package backends

import (
	"context"
	"sync"

	"github.com/sigmavirus24/circuitry"
)

type infoWithLock struct {
	information circuitry.CircuitInformation
	lock        *sync.Mutex
}

// InMemoryBackend defines an in
// [github.com/sigmavirus24/circuitry.StorageBackender] backend designed to be
// used primarily for proofs of concept and testing
type InMemoryBackend struct {
	information map[string]infoWithLock
	lock        sync.Mutex
}

// NewInMemoryBackend creates a new [InMemoryBackend]
func NewInMemoryBackend() circuitry.StorageBackender {
	return &InMemoryBackend{
		information: make(map[string]infoWithLock),
	}
}

func (b *InMemoryBackend) setDefault(name string) infoWithLock {
	i := infoWithLock{circuitry.CircuitInformation{}, &sync.Mutex{}}
	b.information[name] = i
	return i
}

// Store saves the circuit information in memory
func (b *InMemoryBackend) Store(_ context.Context, name string, ci circuitry.CircuitInformation) error {
	b.lock.Lock()
	defer b.lock.Unlock()
	info, ok := b.information[name]
	if !ok {
		info = b.setDefault(name)
	}
	b.information[name] = infoWithLock{information: ci, lock: info.lock}
	return nil
}

// Retrieve fetches the desired circuit state from memory
func (b *InMemoryBackend) Retrieve(_ context.Context, name string) (circuitry.CircuitInformation, error) {
	b.lock.Lock()
	defer b.lock.Unlock()
	info, ok := b.information[name]
	if !ok {
		info = b.setDefault(name)
	}
	return info.information, nil
}

// Lock provides a lock around the given circuit state to protect the
// atomicity of the data
func (b *InMemoryBackend) Lock(_ context.Context, name string) (sync.Locker, error) {
	b.lock.Lock()
	defer b.lock.Unlock()
	info, ok := b.information[name]
	if !ok {
		info = b.setDefault(name)
	}
	return info.lock, nil
}

var _ circuitry.StorageBackender = (*InMemoryBackend)(nil)

// WithInMemoryBackend creates an in memory backend storage for a circuit
// breaker
func WithInMemoryBackend() circuitry.SettingsOption {
	return circuitry.WithStorageBackend(NewInMemoryBackend())
}
