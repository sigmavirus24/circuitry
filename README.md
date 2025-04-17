# circuitry - Distributed Circuit Breakers for Go

If you have a distributed task queue that is multi-tenant and you want circuit
breakers around a tenant's jobs in that queue, existing Go circuit breakers
are only useful if you retry that job in the current worker in the context of
the existing task. However, that can lead to requiring very low numbers of
retries and not being able to backoff each time you retry the task because
those circuit breakers cannot store state elsewhere and retrieve it at the
start. Otherwise, if you use an exponential backoff with a large number of
retries, this one task can consume a great deal of that one worker's time
leading to starvation amongst other tenants' tasks.

## Example Usage

1. Start by choosing a storage provider. Circuitry comes with two providers by
   default:

   * [Redis](./backends/redis/README.md)

   * [DynamoDB](./backends/dynamodb/README.md)

   Once you've configured your chosen backend client, you'll want to set that
   in your Circuit Breaker Factory's Settings

```go
import (
    "errors"
    "fmt"
    "time"

    "github.com/sigmavirus24/circuitry"
    "github.com/sigmavirus24/circuitry/log"
)

func CreateCircuitBreakerFactory(backend circuitry.StorageBackender, logger log.Logger) (*circuitry.CircuitBreakerFactory, error) {
    settings, err := circuitry.NewFactorySettings(
        circuitry.WithStorageBackend(backend),
        circuitry.WithLogger(logger),
        circuitry.WithNameFunc(func(baseName string, circuitContext map[string]any) string {
            tenantId := circuitContext["tenant_id"] // For demonstration purposes, I strongly suggest this be a stable id
            return fmt.Sprintf("%s/%s", name, tenantId)
        })
        circuitry.WithDefaultFallbackErrorMatcher(),
        circuitry.WithFailureCountThreshold(15),
        circuitry.WithCloseThreshold(5),
        circuitry.WithAllowAfter(30 * time.Minute),
        circuitry.WithCyclicClearAfter(12 * time.Hour),
        circuitry.WithStateChangeCallback(func(name string, circuitContext map[string]any, from, to circuitry.CircuitState) {
            logger.
                WithFields(circuitContext). // Ensure no sensitive information is logged here
                WithFields(log.Fields{
                    "name": name,
                    "from": from.String(),
                    "to": to.String(),
                }).Debug("state transition")
        })
    )
    if err != nil {
        return nil, err
    }
    return circuitry.NewCircuitBreakerFactory(settings), nil
}

func TenantContext(id string) map[string]any {
    return map[string]any{
        "tenant_id": id,
        // Include your other context
    }
}

func Work() (any, error) {
    // Do your work here
    return struct{}{}, nil
}

func main() {
    // Setup your backend and logger
    factory := CreateCircuitBreakerFactory(backend, logger)
    tenantIDs := []string{
        // IDs
    }
    for _, tenantID := range tenantIDs {
        tenantCtx := TenantContext(tenantID)
        breaker := factory.BreakerFor("do-work-example", tenantCtx)
        result, workErr, err := breaker.Execute(Work)
        if errors.Is(err, circuitry.ErrCircuitBreakerOpen) {
            logger.WithField("tenant_id", tenantID).Warn("circuit already open")
            continue
        }
        if err != nil {
            logger.WithError(err).WithField("tenant_id", tenantID).Error("circuit breaker could not start work")
        }
        if workErr != nil {
            logger.WithError(err).WithField("tenant_id", tenantID).Error("work function failed")
        }
        logger.WithField("tenant_id", tenantID).Debug("finished work function successfully")
    }
}
```

Some of this has been simplified for demonstration purposes.


## Why make this?

I've had experience in the past with distributed circuit breakers in other
languages. I needed one for a project I'm working on in Go and couldn't find
one that already existed and there did not appear to be anyway to adapt
existing circuit breaker implementations to store and retrieve state from a
remote backend. As a result, I implemented the pattern for myself with great
inspiration from go-breaker as explained in the [Credits](#credits).

## Credits

Much of the design of this library was informed by github.com/sony/go-breaker.
The significant differences are as follows:

* Representation of the internal state of the circuit breaker to allow it to
  be stored/retrieved

* Naming of some constants to make it clearer what they are

* Fix bug in transitions from Half-Open to Open state where generation is
  incremented and counts are reset incorrectly

* Interfaces for storing the state remotely to create a distributed circuit
  breaker

* A single interface rather than two interfaces (e.g., instead of a
  CircuitBreaker and TwoStepCircuitBreaker, one gets a single interface)

* Default implementations for many things

* Options Function style configuration


## Roadmap

* Add logging to core module areas that make most sense

* Ensure that we have good primitives and interfaces

  * Does it make sense to rely on `sync.Locker` or should we have locks with
    heartbeats that can expire if the breaker doesn't keep it alive?

* Ensure that the documentation is sufficient

* Add more examples

* Do we need metrics that this package would expose?
