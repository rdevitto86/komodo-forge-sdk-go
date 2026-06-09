package health

import (
	"context"
	"encoding/json"
	"net/http"
	"sync"
	"time"

	"golang.org/x/sync/singleflight"
)

const (
	defaultCacheTTL     = 10 * time.Second
	defaultCheckTimeout = 3 * time.Second
)

// Probes a single downstream dependency and reports whether it is reachable.
type Checker interface {
	Name() string
	Check(ctx context.Context) error
}

type checkerFunc struct {
	name string
	fn   func(ctx context.Context) error
}

func (c checkerFunc) Name() string                    { return c.name }
func (c checkerFunc) Check(ctx context.Context) error { return c.fn(ctx) }

// Adapts a plain probe function into a Checker under the given dependency name.
func CheckerFunc(name string, fn func(ctx context.Context) error) Checker {
	return checkerFunc{name: name, fn: fn}
}

type cacheEntry struct {
	err       error
	expiresAt time.Time
}

// Configures a readiness handler returned by NewReadyHandler.
type Option func(*readyHandler)

// Overrides how long a checker's result is reused before it is probed again. Defaults to 10s.
func WithCacheTTL(d time.Duration) Option {
	return func(h *readyHandler) { h.cacheTTL = d }
}

// Overrides the per-checker context deadline applied when no deadline is already set on the request context. Defaults to 3s.
func WithCheckTimeout(d time.Duration) Option {
	return func(h *readyHandler) { h.checkTimeout = d }
}

type failingDep struct {
	Dep   string `json:"dep"`
	Error string `json:"error"`
}

type readyHandler struct {
	checkers     []Checker
	cacheTTL     time.Duration
	checkTimeout time.Duration

	mu    sync.RWMutex
	cache map[string]cacheEntry
	sf    singleflight.Group
}

// Builds a GET /health/ready handler that probes every registered Checker, caching each result for
// CacheTTL (default 10s) to absorb load-balancer probe spam. Responds 200 when all dependencies are
// reachable, or 503 with the list of failing dependencies.
func NewReadyHandler(checkers []Checker, opts ...Option) http.HandlerFunc {
	h := &readyHandler{
		checkers:     checkers,
		cacheTTL:     defaultCacheTTL,
		checkTimeout: defaultCheckTimeout,
		cache:        make(map[string]cacheEntry),
	}
	for _, opt := range opts {
		opt(h)
	}
	return h.serveHTTP
}

func (h *readyHandler) serveHTTP(wtr http.ResponseWriter, req *http.Request) {
	var (
		wg      sync.WaitGroup
		mu      sync.Mutex
		failing []failingDep
	)

	for _, checker := range h.checkers {
		wg.Add(1)
		go func(c Checker) {
			defer wg.Done()
			if err := h.run(req.Context(), c); err != nil {
				mu.Lock()
				failing = append(failing, failingDep{Dep: c.Name(), Error: err.Error()})
				mu.Unlock()
			}
		}(checker)
	}
	wg.Wait()

	wtr.Header().Set("Content-Type", "application/json")
	if len(failing) == 0 {
		wtr.WriteHeader(http.StatusOK)
		json.NewEncoder(wtr).Encode(map[string]string{"status": "OK"})
		return
	}

	wtr.WriteHeader(http.StatusServiceUnavailable)
	json.NewEncoder(wtr).Encode(map[string][]failingDep{"failing": failing})
}

// Returns the cached result for the checker if it is still fresh, otherwise probes the dependency
// (deduping concurrent probes of the same dependency via singleflight) and caches the outcome.
func (h *readyHandler) run(ctx context.Context, c Checker) error {
	if cached, ok := h.cached(c.Name()); ok {
		return cached
	}

	res, _, _ := h.sf.Do(c.Name(), func() (any, error) {
		if cached, ok := h.cached(c.Name()); ok {
			return cached, nil
		}

		checkCtx := ctx
		if _, hasDeadline := ctx.Deadline(); !hasDeadline {
			var cancel context.CancelFunc
			checkCtx, cancel = context.WithTimeout(ctx, h.checkTimeout)
			defer cancel()
		}

		checkErr := c.Check(checkCtx)
		h.mu.Lock()
		h.cache[c.Name()] = cacheEntry{err: checkErr, expiresAt: time.Now().Add(h.cacheTTL)}
		h.mu.Unlock()
		return checkErr, nil
	})
	if res == nil {
		return nil
	}
	return res.(error)
}

func (h *readyHandler) cached(name string) (error, bool) {
	h.mu.RLock()
	entry, ok := h.cache[name]
	h.mu.RUnlock()
	if !ok || time.Now().After(entry.expiresAt) {
		return nil, false
	}
	return entry.err, true
}
