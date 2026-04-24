package client

import (
	"errors"
	"sync"
	"testing"
	"time"
)

var errFake = errors.New("fake error")

func cfgWith(threshold, successThreshold int, timeout time.Duration, maxHalfOpen int) Config {
	return Config{
		FailureThreshold:    threshold,
		SuccessThreshold:    successThreshold,
		OpenTimeout:         timeout,
		MaxHalfOpenRequests: maxHalfOpen,
	}
}

func TestBreaker_TripsAfterFailureThreshold(t *testing.T) {
	b := newBreaker(cfgWith(3, 1, time.Minute, 1))
	fail := func() error { return errFake }

	for i := range 2 {
		if err := b.execute("k", fail); !errors.Is(err, errFake) {
			t.Fatalf("call %d: expected errFake, got %v", i, err)
		}
	}

	if err := b.execute("k", fail); !errors.Is(err, errFake) {
		t.Fatalf("threshold call: expected errFake, got %v", err)
	}
	if err := b.execute("k", fail); !errors.Is(err, ErrOpen) {
		t.Fatalf("after trip: expected ErrOpen, got %v", err)
	}
}

func TestBreaker_RejectsWhileOpen(t *testing.T) {
	b := newBreaker(cfgWith(1, 1, time.Hour, 1))
	_ = b.execute("k", func() error { return errFake })

	for i := range 5 {
		if err := b.execute("k", func() error { return nil }); !errors.Is(err, ErrOpen) {
			t.Fatalf("call %d: expected ErrOpen, got %v", i, err)
		}
	}
}

func TestBreaker_TransitionsToHalfOpenAfterTimeout(t *testing.T) {
	b := newBreaker(cfgWith(1, 1, 50*time.Millisecond, 1))
	_ = b.execute("k", func() error { return errFake })

	time.Sleep(60 * time.Millisecond)

	if err := b.execute("k", func() error { return nil }); err != nil {
		t.Fatalf("expected nil after timeout, got %v", err)
	}
}

func TestBreaker_SuccessfulHalfOpenCloses(t *testing.T) {
	b := newBreaker(cfgWith(1, 2, 50*time.Millisecond, 2))
	_ = b.execute("k", func() error { return errFake })

	time.Sleep(60 * time.Millisecond)

	if err := b.execute("k", func() error { return nil }); err != nil {
		t.Fatalf("first half-open success: got %v", err)
	}
	if err := b.execute("k", func() error { return nil }); err != nil {
		t.Fatalf("second half-open success: got %v", err)
	}
	if err := b.execute("k", func() error { return errFake }); !errors.Is(err, errFake) {
		t.Fatalf("after close: expected errFake, got %v", err)
	}
}

func TestBreaker_FailureInHalfOpenReopens(t *testing.T) {
	b := newBreaker(cfgWith(1, 2, 50*time.Millisecond, 1))
	_ = b.execute("k", func() error { return errFake })

	time.Sleep(60 * time.Millisecond)

	if err := b.execute("k", func() error { return errFake }); !errors.Is(err, errFake) {
		t.Fatalf("expected errFake from half-open failure, got %v", err)
	}
	if err := b.execute("k", func() error { return nil }); !errors.Is(err, ErrOpen) {
		t.Fatalf("after reopen: expected ErrOpen, got %v", err)
	}
}

func TestBreaker_OnStateChangeCallback(t *testing.T) {
	var mu sync.Mutex
	var transitions []struct{ from, to BreakerState }

	cfg := Config{
		FailureThreshold:    2,
		SuccessThreshold:    1,
		OpenTimeout:         50 * time.Millisecond,
		MaxHalfOpenRequests: 1,
		OnStateChange: func(key string, from, to BreakerState) {
			mu.Lock()
			transitions = append(transitions, struct{ from, to BreakerState }{from, to})
			mu.Unlock()
		},
	}

	b := newBreaker(cfg)
	_ = b.execute("k", func() error { return errFake })
	_ = b.execute("k", func() error { return errFake }) // trips → Open

	time.Sleep(60 * time.Millisecond)

	_ = b.execute("k", func() error { return nil }) // HalfOpen → Closed

	mu.Lock()
	got := make([]struct{ from, to BreakerState }, len(transitions))
	copy(got, transitions)
	mu.Unlock()

	if len(got) != 3 {
		t.Fatalf("expected 3 transitions, got %d: %v", len(got), got)
	}
	if got[0].from != BreakerClosed || got[0].to != BreakerOpen {
		t.Errorf("transition 0: want Closed→Open, got %v→%v", got[0].from, got[0].to)
	}
	if got[1].from != BreakerOpen || got[1].to != BreakerHalfOpen {
		t.Errorf("transition 1: want Open→HalfOpen, got %v→%v", got[1].from, got[1].to)
	}
	if got[2].from != BreakerHalfOpen || got[2].to != BreakerClosed {
		t.Errorf("transition 2: want HalfOpen→Closed, got %v→%v", got[2].from, got[2].to)
	}
}

func TestBreaker_ConcurrentSafety(t *testing.T) {
	b := newBreaker(cfgWith(10, 2, 50*time.Millisecond, 5))

	const goroutines = 50
	var wg sync.WaitGroup
	wg.Add(goroutines)

	for i := range goroutines {
		go func(i int) {
			defer wg.Done()
			_ = b.execute("k", func() error {
				if i%3 == 0 {
					return errFake
				}
				return nil
			})
		}(i)
	}

	wg.Wait()
}
