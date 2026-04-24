package client

import (
	"errors"
	"sync"
	"time"
)

// ErrOpen is returned by Do when the circuit breaker for the target host is open.
var ErrOpen = errors.New("circuit breaker is open")

// BreakerState represents the current state of a circuit breaker.
type BreakerState int

const (
	BreakerClosed BreakerState = iota
	BreakerOpen
	BreakerHalfOpen
)

func (s BreakerState) String() string {
	switch s {
		case BreakerClosed:
			return "Closed"
		case BreakerOpen:
			return "Open"
		case BreakerHalfOpen:
			return "HalfOpen"
		default:
			return "Unknown"
	}
}

// Config controls the behaviour of the circuit breaker attached via WithCircuitBreaker.
type Config struct {
	// FailureThreshold is the number of failures before the breaker opens.
	// Recommended default: 5.
	FailureThreshold int

	// SuccessThreshold is the number of consecutive successes in HalfOpen
	// required before the breaker closes again.
	// Recommended default: 2.
	SuccessThreshold int

	// OpenTimeout is how long the breaker stays Open before moving to HalfOpen.
	// Recommended default: 60s.
	OpenTimeout time.Duration

	// MaxHalfOpenRequests is the max concurrent requests allowed through in HalfOpen.
	// Recommended default: 1.
	MaxHalfOpenRequests int

	// OnStateChange is called when a breaker for a key transitions state.
	// nil means no-op.
	OnStateChange func(key string, from, to BreakerState)
}

type breakerState struct {
	state            BreakerState
	failures         int
	successes        int
	halfOpenInFlight int
	openedAt         time.Time
}

type breaker struct {
	cfg   Config
	mu    sync.Mutex
	state map[string]*breakerState
}

func newBreaker(cfg Config) *breaker {
	return &breaker{
		cfg:   cfg,
		state: make(map[string]*breakerState),
	}
}

func (b *breaker) stateFor(key string) *breakerState {
	s, ok := b.state[key]
	if !ok {
		s = &breakerState{}
		b.state[key] = s
	}
	return s
}

func (b *breaker) transition(key string, s *breakerState, to BreakerState) {
	from := s.state
	s.state = to
	s.failures = 0
	s.successes = 0
	s.halfOpenInFlight = 0

	if to == BreakerOpen {
		s.openedAt = time.Now()
	}
	if b.cfg.OnStateChange != nil {
		b.cfg.OnStateChange(key, from, to)
	}
}

func (b *breaker) execute(key string, fn func() error) error {
	b.mu.Lock()
	s := b.stateFor(key)

	switch s.state {
		case BreakerOpen:
			if time.Since(s.openedAt) < b.cfg.OpenTimeout {
				b.mu.Unlock()
				return ErrOpen
			}
			b.transition(key, s, BreakerHalfOpen)
			fallthrough

		case BreakerHalfOpen:
			if s.halfOpenInFlight >= b.cfg.MaxHalfOpenRequests {
				b.mu.Unlock()
				return ErrOpen
			}

			s.halfOpenInFlight++
			b.mu.Unlock()

			err := fn()

			b.mu.Lock()
			s.halfOpenInFlight--

			if err != nil {
				b.transition(key, s, BreakerOpen)
				b.mu.Unlock()
				return err
			}

			s.successes++
			if s.successes >= b.cfg.SuccessThreshold {
				b.transition(key, s, BreakerClosed)
			}

			b.mu.Unlock()
			return nil

	// BreakerClosed
	default:
		b.mu.Unlock()

		err := fn()
		if err == nil { return nil }

		b.mu.Lock()
		s.failures++

		if s.failures >= b.cfg.FailureThreshold {
			b.transition(key, s, BreakerOpen)
		}

		b.mu.Unlock()
		return err
	}
}
