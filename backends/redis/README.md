# Redis Backend for Circuitry

This provides a StorageBackender implementation for circuitry that uses Redis
as the backend.

_Note_: This requires `github.com/redis/go-redis/v9`

## Usage

```golang
import (
    "fmt"
    "time"

    "github.com/bsm/redislock"
    "github.com/redis/go-redis/v9"
    "github.com/sigmavirus24/circuitry"
    redisbackend "github.com/sigmavirus24/circuitry/backends/redis"
)

func main() {
    settings, err := circuitry.NewFactorySettings(
        redisbackend.WithRedisBackend(
            &redis.Options{
                // See also https://pkg.go.dev/github.com/redis/go-redis/v9#readme-quickstart
                // and https://pkg.go.dev/github.com/redis/go-redis/v9#Options
                Addr: "localhost:6379",
                Password: "",
                DB: 0,
            },
            &redislock.Options{
                RetryStrategy: redislock.NoRetry(),
            },
            1 * time.Hour,
        ),
        circuitry.WithDefaultNameFunc(),
        circuitry.WithDefaultTripFunc(),
        circuitry.WithDefaultFallbackErrorMatcher(),
    )
    if err != nil {
        fmt.Printf("could not create settings: %v\n", err)
        return
    }
    factory := circuitry.NewCircuitBreakerFactory(settings)
    breaker := factory.BreakerFor("my-name", map[string]any{})
    ctx := context.Background()
    err = breaker.Start(ctx)
    if err != nil {
        fmt.Printf("could not start circuit breaker: %v\n", err)
    }
    defer breaker.End(ctx, nil)
}
```
