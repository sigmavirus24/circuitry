# Circuitry Backends

This package provides a built-in in-memory backend for use with Circuitry,
which can be useful for testing or proofs of concept.

It also has other sub-packages which provide implementations for:

* Redis
* DynamoDB


## Quickstart

```go
import (
    "context"
    "fmt"

    "github.com/google/uuid"
    "github.com/sigmavirus24/circuitry"
    "github.com/sigmavirus24/circuitry/backends"
)

func NameGenerator(name string, breakerContext map[string]any) string {
    workID := breakerContext["work_uuid"]
    return fmt.Sprintf("%s::%s", name, workID)
}

func BreakerFactory() *circuitry.CircuitBreakerFactory {
    settings, err := circuitry.NewFactorySettings(
        backends.WithInMemoryBackend(),
        circuitry.WithNameFunc
        // Additional options
    )
    return circuitry.NewCircuitBreakerFactory(settings)
}

func Work() (any, error) {
    // Function with the work you want to protect with the circuit breaker
}

func main() {
    ctx := context.Background()
    factory := BreakerFactory()
    breakerContext := map[string]string{
        "work_uuid": uuid.NewString(),
        // Additional context relevant
    }
    breaker := factory.BreakerFor("example-breaker", breakerContext)
    workResult, workErr, breakerErr := breaker.Execute(ctx, Work)
    if breakerErr != nil {
        fmt.Printf("could not execute work in breaker: %v\n", breakerErr)
        return
    }
    if workErr != nil {
        fmt.Printf("work failed: %s\n", workErr)
        state, err := breaker.State(ctx)
        if err != nil {
            fmt.Printf("cannot display breaker state: %s\n", err)
            return
        }
        fmt.Printf("breaker status: %s\n", state)
        return
    }
    fmt.Printf("finished work")
}
```
