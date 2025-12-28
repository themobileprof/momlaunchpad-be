package circuitbreaker

import (
	"errors"
	"sync"
	"time"
)

var (
	ErrCircuitOpen     = errors.New("circuit breaker is open")
	ErrTooManyRequests = errors.New("too many requests")
)

// State represents circuit breaker state
type State int

const (
	StateClosed State = iota
	StateHalfOpen
	StateOpen
)

func (s State) String() string {
	switch s {
	case StateClosed:
		return "closed"
	case StateHalfOpen:
		return "half-open"
	case StateOpen:
		return "open"
	default:
		return "unknown"
	}
}

// CircuitBreaker implements the circuit breaker pattern
type CircuitBreaker struct {
	maxFailures  int
	resetTimeout time.Duration

	mu              sync.RWMutex
	state           State
	failures        int
	successes       int
	lastFailureTime time.Time
	lastStateChange time.Time
}

// NewCircuitBreaker creates a new circuit breaker
func NewCircuitBreaker(maxFailures int, resetTimeout time.Duration) *CircuitBreaker {
	return &CircuitBreaker{
		maxFailures:     maxFailures,
		resetTimeout:    resetTimeout,
		state:           StateClosed,
		lastStateChange: time.Now(),
	}
}

// Call executes a function with circuit breaker protection
func (cb *CircuitBreaker) Call(fn func() error) error {
	if err := cb.beforeCall(); err != nil {
		return err
	}

	err := fn()
	cb.afterCall(err)

	return err
}

// beforeCall checks if call is allowed
func (cb *CircuitBreaker) beforeCall() error {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	now := time.Now()

	switch cb.state {
	case StateClosed:
		return nil

	case StateOpen:
		// Check if we should transition to half-open
		if now.Sub(cb.lastFailureTime) > cb.resetTimeout {
			cb.state = StateHalfOpen
			cb.lastStateChange = now
			cb.successes = 0
			return nil
		}
		return ErrCircuitOpen

	case StateHalfOpen:
		// Allow limited requests in half-open state
		if cb.successes >= 1 {
			return ErrTooManyRequests
		}
		return nil
	}

	return nil
}

// afterCall updates circuit breaker state after call
func (cb *CircuitBreaker) afterCall(err error) {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	now := time.Now()

	if err != nil {
		cb.failures++
		cb.lastFailureTime = now

		switch cb.state {
		case StateClosed:
			if cb.failures >= cb.maxFailures {
				cb.state = StateOpen
				cb.lastStateChange = now
			}

		case StateHalfOpen:
			cb.state = StateOpen
			cb.lastStateChange = now
		}
	} else {
		cb.successes++

		switch cb.state {
		case StateHalfOpen:
			// Successful call in half-open, close the circuit
			cb.state = StateClosed
			cb.lastStateChange = now
			cb.failures = 0
			cb.successes = 0

		case StateClosed:
			// Reset failure count on success
			if now.Sub(cb.lastFailureTime) > cb.resetTimeout {
				cb.failures = 0
			}
		}
	}
}

// State returns current circuit breaker state
func (cb *CircuitBreaker) State() State {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	return cb.state
}

// Stats returns circuit breaker statistics
func (cb *CircuitBreaker) Stats() (state State, failures, successes int) {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	return cb.state, cb.failures, cb.successes
}

// Reset resets the circuit breaker to closed state
func (cb *CircuitBreaker) Reset() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.state = StateClosed
	cb.failures = 0
	cb.successes = 0
	cb.lastStateChange = time.Now()
}
