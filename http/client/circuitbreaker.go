package client

import (
	"errors"
	"sync"
	"time"
)

var ErrOpen = errors.New("circuit breaker is open")

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

// Controls the behaviour of the circuit breaker attached via ClientConfig.CircuitBreaker.
type BreakerConfig struct {
	FailureThreshold    int                                     // failures before the breaker opens; recommended default 5
	SuccessThreshold    int                                     // consecutive HalfOpen successes needed to close; recommended default 2
	OpenTimeout         time.Duration                           // time Open before moving to HalfOpen; recommended default 60s
	MaxHalfOpenRequests int                                     // max concurrent requests allowed through in HalfOpen; recommended default 1
	MaxHosts            int                                     // caps tracked hosts; new hosts fail open past the cap; 0 applies DefaultMaxHosts, negative means unlimited
	OnStateChange       func(key string, from, to BreakerState) // called on a key's state transition; nil means no-op
}

type breakerState struct {
	state            BreakerState
	failures         int
	successes        int
	halfOpenInFlight int
	openedAt         time.Time
}

// Pairs per-host breaker state with its own mutex so hosts don't contend.
type breakerEntry struct {
	mu    sync.Mutex
	state breakerState
}

type breaker struct {
	cfg       BreakerConfig
	hosts     sync.Map // string → *breakerEntry
	hostCount int      // approximate; protected by hostMu
	hostMu    sync.Mutex
}

// DefaultMaxHosts bounds the per-host breaker map when BreakerConfig.MaxHosts is left at
// zero, so a client fanning out to many hosts cannot grow the map without limit. Set
// MaxHosts negative to opt back into unlimited tracking.
const DefaultMaxHosts = 1024

func newBreaker(cfg BreakerConfig) *breaker {
	if cfg.MaxHosts == 0 {
		cfg.MaxHosts = DefaultMaxHosts
	}
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

// Removes closed, zero-failure entries from the host map to reclaim memory for hosts that are no longer failing.
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
