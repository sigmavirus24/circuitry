package redis

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"time"

	"github.com/bsm/redislock"
	redis "github.com/redis/go-redis/v9"
	"github.com/sigmavirus24/circuitry"
)

// Client describes the interface expected for this backend to function
// appropriately
type Client interface {
	Get(context.Context, string) *redis.StringCmd
	Set(context.Context, string, any, time.Duration) *redis.StatusCmd
	SetArgs(context.Context, string, any, redis.SetArgs) *redis.StatusCmd
	redis.Scripter // Need this interface for redislock
}

// Locker describes the interface expected for this backend to function
// appropriately
type Locker interface {
	Obtain(context.Context, string, time.Duration, *redislock.Options) (*redislock.Lock, error)
}

type redLock struct {
	ctx  context.Context
	lock *redislock.Lock
}

func (l *redLock) Lock()   {}
func (l *redLock) Unlock() { l.lock.Release(l.ctx) }

// Backend implements the StorageBackender interface for Redis using
// redis/go-redis/v9 and bsm/redislock
type Backend struct {
	Client         Client
	Locker         Locker
	LockOpts       *redislock.Options
	DefaultLockTTL time.Duration
}

// Store saves the CircuitInformation in Redis under the named key after
// seriailizing it to JSON.
func (c *Backend) Store(ctx context.Context, name string, ci circuitry.CircuitInformation) error {
	bytes, _ := json.Marshal(ci) // We know CircuitInformation is Marshal-able
	cmd := c.Client.SetArgs(ctx, name, string(bytes), redis.SetArgs{ExpireAt: ci.ExpiresAfter})
	if err := cmd.Err(); err != nil {
		return err
	}
	return nil
}

// Retrieve looks up the key in Redis and returns the value after
// deserializing it from JSON. If the key is not present in Redis, this will
// return an empty circuitry.CircuitInformation
func (c *Backend) Retrieve(ctx context.Context, name string) (circuitry.CircuitInformation, error) {
	jsonCI, err := c.Client.Get(ctx, name).Result()
	ci := circuitry.CircuitInformation{}
	if err != nil {
		if errors.Is(err, redis.Nil) {
			// If the name is not in Redis, we should not return that error,
			// but instead an empty CircuitInformation
			return ci, nil
		}
		return ci, err
	}
	err = json.Unmarshal([]byte(jsonCI), &ci)
	if err != nil {
		return circuitry.CircuitInformation{}, err
	}
	return ci, nil
}

// Lock builds a lock in Redis with the DefaultTTL and returns an interface
// that matches sync.Locker
func (c *Backend) Lock(ctx context.Context, name string) (sync.Locker, error) {
	lock, err := c.Locker.Obtain(ctx, name, c.DefaultLockTTL, c.LockOpts)
	if err != nil {
		return nil, err
	}
	return &redLock{ctx, lock}, nil
}

// New builds a new StorageBackender for circuitry.
func New(clientOpts *redis.Options, lockOpts *redislock.Options, defaultLockTTL time.Duration) circuitry.StorageBackender {
	redClient := redis.NewClient(clientOpts)
	locker := redislock.New(redClient)
	return &Backend{redClient, locker, lockOpts, defaultLockTTL}
}

// WithRedisBackend provides a way to configure the StorageBackend for a
// Circuit Breaker Factory's esttings.
func WithRedisBackend(clientOpts *redis.Options, lockOpts *redislock.Options, defaultLockTTL time.Duration) circuitry.SettingsOption {
	backend := New(clientOpts, lockOpts, defaultLockTTL)
	return circuitry.WithStorageBackend(backend)
}
