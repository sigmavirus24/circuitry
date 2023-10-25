# circuitry - Distributed Circuit Breakers for Go

## Credits

Much of the design of this library was informed by github.com/sony/go-breaker.
The significant differences are as follows:

* Representation of the internal state of the circuit breaker to allow it to
  be stored/retrieved 

* Naming of some constants to make it clearer what they are

* Interfaces for storing the state remotely to create a distributed circuit
  breaker

* A single interface rather than two interfaces (e.g., instead of a
  CircuitBreaker and TwoStepCircuitBreaker, one gets a single interface)

* Default implementations for many things

* Options Function style configuration
