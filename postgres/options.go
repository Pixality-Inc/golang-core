package postgres

import "github.com/pixality-inc/golang-core/circuit_breaker"

type Option func(database *DatabaseImpl)

func MaxPoolSize(size int) Option {
	return func(db *DatabaseImpl) {
		db.maxPoolSize = size
	}
}

func WithCircuitBreaker(cb circuit_breaker.CircuitBreaker) Option {
	return func(db *DatabaseImpl) {
		db.circuitBreaker = cb
	}
}
