package idempotency

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"
)

// mirrors Redis's atomicity guarantees (SETNX/EXISTS/DEL under one lock) for component tests
type fakeRedisAPI struct {
	mu    sync.Mutex
	store map[string]string
}

func (f *fakeRedisAPI) ensure() map[string]string {
	if f.store == nil {
		f.store = make(map[string]string)
	}
	return f.store
}

func (f *fakeRedisAPI) Get(ctx context.Context, key string) (string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	v, ok := f.ensure()[key]
	if !ok {
		return "", fmt.Errorf("not found")
	}
	return v, nil
}

func (f *fakeRedisAPI) Set(ctx context.Context, key, value string, ttl int64) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.ensure()[key] = value
	return nil
}

func (f *fakeRedisAPI) Delete(ctx context.Context, key string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	delete(f.ensure(), key)
	return nil
}

func (f *fakeRedisAPI) Incr(ctx context.Context, key string) (int64, error) {
	return 0, fmt.Errorf("not implemented")
}

func (f *fakeRedisAPI) Expire(ctx context.Context, key string, ttl int64) error {
	return nil
}

func (f *fakeRedisAPI) SetNX(ctx context.Context, key, value string, ttl int64) (bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if _, exists := f.ensure()[key]; exists {
		return false, nil
	}
	f.ensure()[key] = value
	return true, nil
}

func (f *fakeRedisAPI) Exists(ctx context.Context, key string) (bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	_, ok := f.ensure()[key]
	return ok, nil
}

func (f *fakeRedisAPI) Ping(ctx context.Context) error { return nil }
func (f *fakeRedisAPI) Close() error                   { return nil }

func (f *fakeRedisAPI) AllowDistributed(ctx context.Context, key string, rate, burst float64, ttlSec int) (bool, time.Duration, error) {
	return true, 0, nil
}

func TestDistributedCache_StoreIfAbsent_FirstWriterWins(t *testing.T) {
	cache := NewDistributedCache(&fakeRedisAPI{})

	stored, err := cache.StoreIfAbsent("key", int64(1), 60)
	if err != nil || !stored {
		t.Fatalf("expected first StoreIfAbsent to succeed, got stored=%v err=%v", stored, err)
	}

	stored, err = cache.StoreIfAbsent("key", int64(1), 60)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if stored {
		t.Error("expected second StoreIfAbsent for the same key to report not-stored")
	}
}

func TestDistributedCache_LoadAndDelete(t *testing.T) {
	cache := NewDistributedCache(&fakeRedisAPI{})

	if _, ok := cache.Load("missing"); ok {
		t.Error("expected Load to report absent for a key never stored")
	}

	if err := cache.Store("present", int64(1), 60); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := cache.Load("present"); !ok {
		t.Error("expected Load to report present after Store")
	}

	cache.Delete("present")
	if _, ok := cache.Load("present"); ok {
		t.Error("expected Load to report absent after Delete")
	}
}

func TestStore_CheckAndSet_ConcurrentDuplicates_DistributedCache(t *testing.T) {
	store := NewDistributedStore(&fakeRedisAPI{}, 60)

	const callers = 25
	var wg sync.WaitGroup
	results := make([]bool, callers)

	wg.Add(callers)
	for i := range callers {
		go func(i int) {
			defer wg.Done()
			allowed, err := store.CheckAndSet("race-key")
			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}
			results[i] = allowed
		}(i)
	}
	wg.Wait()

	winners := 0
	for _, allowed := range results {
		if allowed {
			winners++
		}
	}
	if winners != 1 {
		t.Errorf("expected exactly one winner among %d concurrent CheckAndSet calls, got %d", callers, winners)
	}
}

func TestStore_CheckAndSet_ConcurrentDuplicates_LocalCache(t *testing.T) {
	store := NewStore("local", 60)

	const callers = 25
	var wg sync.WaitGroup
	results := make([]bool, callers)

	wg.Add(callers)
	for i := range callers {
		go func(i int) {
			defer wg.Done()
			allowed, err := store.CheckAndSet("race-key-local")
			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}
			results[i] = allowed
		}(i)
	}
	wg.Wait()

	winners := 0
	for _, allowed := range results {
		if allowed {
			winners++
		}
	}
	if winners != 1 {
		t.Errorf("expected exactly one winner among %d concurrent CheckAndSet calls, got %d", callers, winners)
	}
}
