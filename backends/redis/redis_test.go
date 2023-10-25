package redis_test

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/bsm/redislock"
	redismock "github.com/go-redis/redismock/v9"
	"github.com/redis/go-redis/v9"

	"github.com/sigmavirus24/circuitry"
	redisbackend "github.com/sigmavirus24/circuitry/backends/redis"
)

func requireExpectations(t *testing.T, mock redismock.ClientMock) {
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("expected expectations to be met, but %v", err)
	}
}

func TestBackendRetrieveMissingKey(t *testing.T) {
	db, mock := redismock.NewClientMock()

	key := "circuit-breaker-1234"
	mock.ExpectGet(key).RedisNil()
	b := redisbackend.Backend{
		Client:         db,
		Locker:         redislock.New(db),
		LockOpts:       &redislock.Options{},
		DefaultLockTTL: 0,
	}
	ci, err := b.Retrieve(context.TODO(), key)
	if err != nil {
		t.Fatalf("expected err to be nil, got %v", err)
	}
	if ci.Generation != 0 {
		t.Fatalf("expected redismock to not have data, but it did, %+v", ci)
	}
	requireExpectations(t, mock)
}

func TestBackendRetrieveError(t *testing.T) {
	db, mock := redismock.NewClientMock()

	key := "circuit-breaker-1234"
	mock.ExpectGet(key).SetErr(redis.ErrClosed)
	b := redisbackend.Backend{
		Client:         db,
		Locker:         redislock.New(db),
		LockOpts:       &redislock.Options{},
		DefaultLockTTL: 0,
	}
	_, err := b.Retrieve(context.TODO(), key)
	if !errors.Is(err, redis.ErrClosed) {
		t.Fatalf("expected err to be redis.ErrClosed(%v), got %v", redis.ErrClosed, err)
	}
	requireExpectations(t, mock)
}

func TestBackendRetrieveWithData(t *testing.T) {
	db, mock := redismock.NewClientMock()
	expectedInfo := circuitry.CircuitInformation{
		Generation:           1,
		ConsecutiveSuccesses: 10,
		Total:                25,
		TotalSuccesses:       17,
		TotalFailures:        18,
	}
	jsonBytes, err := json.Marshal(expectedInfo)
	if err != nil {
		t.Fatalf("serializing CircuitInformation to JSON shouldn't fail but it did, %v", err)
	}

	key := "circuit-breaker-1234"
	mock.ExpectGet(key).SetVal(string(jsonBytes))
	b := redisbackend.Backend{
		Client:         db,
		Locker:         redislock.New(db),
		LockOpts:       &redislock.Options{},
		DefaultLockTTL: 0,
	}
	ci, err := b.Retrieve(context.TODO(), key)
	if err != nil {
		t.Fatalf("expected err to be nil, got %v", err)
	}
	if ci != expectedInfo {
		t.Fatalf("CircuitInformation did not round-trip appropriately, expected %+v, got %+v", expectedInfo, ci)
	}
	requireExpectations(t, mock)
}

func TestBackendRetrieveWithCorruptedData(t *testing.T) {
	db, mock := redismock.NewClientMock()
	expectedInfo := circuitry.CircuitInformation{
		Generation:           1,
		ConsecutiveSuccesses: 10,
		Total:                25,
		TotalSuccesses:       17,
		TotalFailures:        18,
	}
	jsonBytes, err := json.Marshal(expectedInfo)
	if err != nil {
		t.Fatalf("serializing CircuitInformation to JSON shouldn't fail but it did, %v", err)
	}

	key := "circuit-breaker-1234"
	mock.ExpectGet(key).SetVal(string(jsonBytes[1:]))
	b := redisbackend.Backend{
		Client:         db,
		Locker:         redislock.New(db),
		LockOpts:       &redislock.Options{},
		DefaultLockTTL: 0,
	}
	_, err = b.Retrieve(context.TODO(), key)
	if err == nil {
		t.Fatal("expected err to not be nil, got nil")
	}
	requireExpectations(t, mock)
}

func TestBackendStore(t *testing.T) {
	db, mock := redismock.NewClientMock()
	expectedInfo := circuitry.CircuitInformation{Generation: 1, Total: 1, ConsecutiveFailures: 1, TotalFailures: 1, State: circuitry.CircuitOpen}
	jsonBytes, err := json.Marshal(expectedInfo)
	if err != nil {
		t.Fatalf("serializing CircuitInformation to JSON shouldn't fail but it did, %v", err)
	}
	key := "store-circuit-breaker-info-1234"
	var zero time.Time
	mock.ExpectSetArgs(key, string(jsonBytes), redis.SetArgs{ExpireAt: zero}).SetVal("")

	b := redisbackend.Backend{Client: db, Locker: redislock.New(db), LockOpts: &redislock.Options{}, DefaultLockTTL: 0}
	err = b.Store(context.TODO(), key, expectedInfo)
	if err != nil {
		t.Fatalf("expected successful storage of circuit state but got %v", err)
	}
	requireExpectations(t, mock)
}

func TestBackendStoreError(t *testing.T) {
	db, mock := redismock.NewClientMock()
	expectedInfo := circuitry.CircuitInformation{Generation: 1, Total: 1, ConsecutiveFailures: 1, TotalFailures: 1, State: circuitry.CircuitOpen}
	jsonBytes, err := json.Marshal(expectedInfo)
	if err != nil {
		t.Fatalf("serializing CircuitInformation to JSON shouldn't fail but it did, %v", err)
	}
	key := "store-circuit-breaker-info-1234"
	mock.ExpectSet(key, string(jsonBytes), 0).SetErr(redis.ErrClosed)

	b := redisbackend.Backend{Client: db, Locker: redislock.New(db), LockOpts: &redislock.Options{}, DefaultLockTTL: 0}
	err = b.Store(context.TODO(), key, expectedInfo)
	if err == nil {
		t.Fatal("expected error storing circuit state but got nil")
	}
	requireExpectations(t, mock)
}

func TestBackendLock(t *testing.T) {
	db, mock := redismock.NewClientMock()

	key := "lock-circuit-breaker-1234"
	lockOpts := &redislock.Options{}

	mock.Regexp().ExpectEvalSha(`.*`, []string{key}, `.*`, `[0-9]+`, `0.*`).SetVal("")

	b := redisbackend.Backend{
		Client:         db,
		Locker:         redislock.New(db),
		LockOpts:       lockOpts,
		DefaultLockTTL: 0,
	}
	lock, err := b.Lock(context.TODO(), key)
	if err != nil {
		t.Fatalf("expected to retrieve a lock successfully, but didn't due to %v", err)
	}
	lock.Lock() // Technically a no-op because redislock creates the lock so no point in calling Lock, but we implemented it for sync.Locker
	defer lock.Unlock()

	requireExpectations(t, mock)
}

func TestBackendLockError(t *testing.T) {
	db, mock := redismock.NewClientMock()

	key := "lock-circuit-breaker-1234"
	lockOpts := &redislock.Options{}

	mock.Regexp().ExpectEvalSha(`.*`, []string{key}, `.*`, `[0-9]+`, `0.*`).SetErr(redis.ErrClosed)

	b := redisbackend.Backend{
		Client:         db,
		Locker:         redislock.New(db),
		LockOpts:       lockOpts,
		DefaultLockTTL: 0,
	}
	_, err := b.Lock(context.TODO(), key)
	if !errors.Is(err, redis.ErrClosed) {
		t.Fatalf("expected to get redis.ErrClosed; got err = %v", err)
	}

	requireExpectations(t, mock)
}

func TestNewBackend(t *testing.T) {
	b := redisbackend.New(&redis.Options{
		Addr:     "localhost:6379",
		Password: "",
		DB:       0,
	}, &redislock.Options{RetryStrategy: redislock.NoRetry()}, time.Duration(0))
	switch bType := b.(type) {
	case *redisbackend.Backend:
		return
	default:
		t.Fatalf("expected backend to be *redisbackend.Backend; got %T", bType)
	}
}

func TestWitRedisBackend(t *testing.T) {
	s, err := circuitry.NewFactorySettings(redisbackend.WithRedisBackend(&redis.Options{}, &redislock.Options{}, time.Duration(0)))
	if err != nil {
		t.Fatalf("expected to successfully create FactorySettings; got err = %v", err)
	}
	switch bType := s.StorageBackend.(type) {
	case *redisbackend.Backend:
		return
	default:
		t.Fatalf("expected FactorySettings.StorageBackend to be *redisbackend.Backend; got %T", bType)
	}
}

func TestWithRedisBackend(t *testing.T) {
	optioner := redisbackend.WithRedisBackend(&redis.Options{}, &redislock.Options{}, time.Duration(0))
	_, err := circuitry.NewFactorySettings(optioner, optioner)
	if !errors.Is(err, circuitry.ErrStorageBackendAlreadySet) {
		t.Fatalf("expected to get an ErrSettingConflict; got err = %v", err)
	}
}
