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

	// MaxHosts caps the number of tracked hosts. When the cap is reached, new
	// hosts bypass the breaker entirely (fail open). 0 means unlimited.
	MaxHosts int

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

// breakerEntry pairs per-host state with its own mutex so hosts don't
// contend with each other.
type breakerEntry struct {
	mu    sync.Mutex
	state breakerState
}

type breaker struct {
	cfg      Config
	hosts    sync.Map // string → *breakerEntry
	hostCount int     // approximate; protected by hostMu
	hostMu   sync.Mutex
}

func newBreaker(cfg Config) *breaker {
	return &breaker{cfg: cfg}
}

func (b *breaker) entryFor(key string) (*breakerEntry, bool) {
	if v, ok := b.hosts.Load(key); ok {
		return v.(*breakerEntry), true
	}

	// Check host cap before creating a new entry.
	if b.cfg.MaxHosts > 0 {
		b.hostMu.Lock()
		if b.hostCount >= b.cfg.MaxHosts {
			b.hostMu.Unlock()
			return nil, false // caller must bypass breaker
		}
		b.hostCount++
		b.hostMu.Unlock()
	}

	entry := &breakerEntry{}
	actual, loaded := b.hosts.LoadOrStore(key, entry)
	if loaded {
		// Another goroutine stored first — decrement the counter we pre-incremented.
		if b.cfg.MaxHosts > 0 {
			b.hostMu.Lock()
			b.hostCount--
			b.hostMu.Unlock()
		}
	}
	return actual.(*breakerEntry), true
}

func (b *breaker) transition(key string, e *breakerEntry, to BreakerState) {
	from := e.state.state
	e.state.state = to
	e.state.failures = 0
	e.state.successes = 0
	e.state.halfOpenInFlight = 0

	if to == BreakerOpen {
		e.state.openedAt = time.Now()
	}
	if b.cfg.OnStateChange != nil {
		b.cfg.OnStateChange(key, from, to)
	}
}

func (b *breaker) execute(key string, fn func() error) error {
	entry, ok := b.entryFor(key)
	if !ok {
		// Host cap exceeded — bypass breaker.
		return fn()
	}

	entry.mu.Lock()
	s := &entry.state

	switch s.state {
	case BreakerOpen:
		if time.Since(s.openedAt) < b.cfg.OpenTimeout {
			entry.mu.Unlock()
			return ErrOpen
		}
		b.transition(key, entry, BreakerHalfOpen)
		fallthrough

	case BreakerHalfOpen:
		if s.halfOpenInFlight >= b.cfg.MaxHalfOpenRequests {
			entry.mu.Unlock()
			return ErrOpen
		}
		s.halfOpenInFlight++
		entry.mu.Unlock()

		err := fn()

		entry.mu.Lock()
		s.halfOpenInFlight--
		if err != nil {
			b.transition(key, entry, BreakerOpen)
			entry.mu.Unlock()
			return err
		}
		s.successes++
		if s.successes >= b.cfg.SuccessThreshold {
			b.transition(key, entry, BreakerClosed)
		}
		entry.mu.Unlock()
		return nil

	// BreakerClosed
	default:
		entry.mu.Unlock()

		err := fn()
		if err == nil {
			return nil
		}

		entry.mu.Lock()
		s.failures++
		if s.failures >= b.cfg.FailureThreshold {
			b.transition(key, entry, BreakerOpen)
		}
		entry.mu.Unlock()
		return err
	}
}

// Prune removes closed, zero-failure entries from the host map, reclaiming
// memory for hosts that are no longer actively failing. Call periodically
// from application code if the service contacts many distinct hosts.
func (b *breaker) Prune() {
	b.hosts.Range(func(k, v any) bool {
		e := v.(*breakerEntry)
		e.mu.Lock()
		removable := e.state.state == BreakerClosed && e.state.failures == 0
		e.mu.Unlock()
		if removable {
			b.hosts.Delete(k)
			if b.cfg.MaxHosts > 0 {
				b.hostMu.Lock()
				b.hostCount--
				b.hostMu.Unlock()
			}
		}
		return true
	})
}
