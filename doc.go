// Package circuitry provides a framework for building distribuited Circuit
// Breakers. It's primarily designed to be able to synchronize state with a
// backend so that a distributed system can have one Circuit Breaker for any
// piece it cares to implement that for.
//
// For example, in a system calling any number of other systems, it can be
// beneficial to the health of the first system to implement a circuit breaker
// around calls to the other dependencies they have. If you have N
// dependencies on other systems, you could have N circuit breakers that would
// work across all of the instances of your system to protect it as a whole.
// If instead you had in memory only circuit breakers, each instance of that
// system would have to individually reach your failure threshold to trip the
// breaker before all instances have stopped talking to it. That could affect
// availability and performance for your users. Alternatively, you might have
// ot set the threshold artificially low and risk the breaker opening when it
// didn't need to.
//
// circuitry aims to provide an excellent distributed circuit breaker pattern
// as well as supportive libraries and a handful of curated backends. For
// example, circuitry provides interfaces for structured logging libraries as
// well as implementations for logrus and slog. Likewise, it allows for
// metrics to be gathered from it by providing interfaces for users to specify
// their sink. Finally, it has a backend interface and provides
// implementations in Redis and DynamoDB for production usage and for
// reference.
//
// # Usage
//
// The main way to interact with circuitry is to create a
// CircuitBreakerFactory. This looks like:
//
//	import (
//		"log/slog"
//		"time"
//
//		"github.com/sigmavirus24/circuitry"
//	)
//
//	logger := slog.Default()
//	settings, err := circuitry.NewFactorySettings(
//		circuitry.WithDefaultTripFunc(),
//		circuitry.WithDefaultNameFunc(),
//		circuitry.WithFailureCountThreshold(15),
//		circuitry.WithAllowAfter(15 * time.Minute),
//		circuitry.WithCyclicClearAfter(1 * time.Hour),
//	)
//	if err != nil {
//		logger.With("err", err).Error("could not instantiate factory settings")
//	}
//	factory := circuitry.NewCircuitBreakerFactory(settings)
//
// Once you have a factory you can create any number of named
// CircuitBreakers. circuitry also allows you to include relevant context when
// creating a CircuitBreaker. For example, maybe you have tracing context you
//
// # Additional Resources
//
// For additional information see also:
// - https://learn.microsoft.com/en-us/previous-versions/msp-n-p/dn589784(v=pandp.10)?redirectedfrom=MSDN
// - https://www.redhat.com/architect/circuit-breaker-architecture-pattern
// - https://docs.aws.amazon.com/prescriptive-guidance/latest/cloud-design-patterns/circuit-breaker.html
package circuitry
