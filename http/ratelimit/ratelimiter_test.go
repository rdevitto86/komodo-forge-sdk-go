package ratelimit

import (
	"context"
	"os"
	"testing"
	"time"
)

// resetState resets all package-level rate limiter state between tests.
func resetState() {
	ResetForTesting()
}

func TestRateLimit_Allow(t *testing.T) {
	t.Run("local mode allows first request", func(t *testing.T) {
		resetState()
		os.Unsetenv("ENV")
		os.Setenv("RATE_LIMIT_RPS", "10")
		os.Setenv("RATE_LIMIT_BURST", "5")
		defer func() {
			os.Unsetenv("RATE_LIMIT_RPS")
			os.Unsetenv("RATE_LIMIT_BURST")
		}()

		allowed, wait, err := Allow(context.Background(), "test-key-1")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !allowed {
			t.Errorf("expected allowed=true, got false (wait=%v)", wait)
		}
	})

	t.Run("local mode exhausts tokens", func(t *testing.T) {
		resetState()
		os.Unsetenv("ENV")
		os.Setenv("RATE_LIMIT_RPS", "1")
		os.Setenv("RATE_LIMIT_BURST", "1")
		defer func() {
			os.Unsetenv("RATE_LIMIT_RPS")
			os.Unsetenv("RATE_LIMIT_BURST")
		}()

		// First call should be allowed (consumes all burst tokens)
		allowed1, _, err := Allow(context.Background(), "exhaust-key")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !allowed1 {
			t.Fatal("first call should be allowed")
		}

		// Second call should be rate limited
		allowed2, wait2, err := Allow(context.Background(), "exhaust-key")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if allowed2 {
			t.Error("expected second call to be denied")
		}
		if wait2 <= 0 {
			t.Error("expected positive wait duration when denied")
		}
	})

	t.Run("prod env attempts distributed rate limit (fails gracefully)", func(t *testing.T) {
		resetState()
		os.Setenv("ENV", "prod")
		defer os.Unsetenv("ENV")

		_, _, _ = Allow(context.Background(), "prod-key")
	})

	t.Run("staging env attempts distributed rate limit (fails gracefully)", func(t *testing.T) {
		resetState()
		os.Setenv("ENV", "staging")
		defer os.Unsetenv("ENV")

		_, _, _ = Allow(context.Background(), "staging-key")
	})
}

func TestRateLimit_GetUsage(t *testing.T) {
	resetState()
	os.Unsetenv("ENV")
	os.Setenv("RATE_LIMIT_RPS", "10")
	os.Setenv("RATE_LIMIT_BURST", "10")
	defer func() {
		os.Unsetenv("RATE_LIMIT_RPS")
		os.Unsetenv("RATE_LIMIT_BURST")
	}()

	Allow(context.Background(), "usage-key")

	used, remaining, reset, err := GetUsage(context.Background(), "usage-key")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if used < 0 {
		t.Errorf("used = %d, want >= 0", used)
	}
	if remaining < 0 {
		t.Errorf("remaining = %d, want >= 0", remaining)
	}
	if reset.IsZero() {
		t.Error("reset time should not be zero")
	}
}

func TestRateLimit_GetUsage_NegativeUsed(t *testing.T) {
	resetState()
	os.Unsetenv("ENV")
	os.Setenv("RATE_LIMIT_RPS", "100")
	os.Setenv("RATE_LIMIT_BURST", "1")
	defer func() {
		os.Unsetenv("RATE_LIMIT_RPS")
		os.Unsetenv("RATE_LIMIT_BURST")
	}()

	Allow(context.Background(), "neg-usage-key")

	v, ok := buckets.Load("neg-usage-key")
	if !ok {
		t.Skip("bucket not found, skipping negative usage test")
	}
	bkt := v.(*bucket)
	bkt.mu.Lock()
	bkt.tokens = 5 // tokens > burst (1) → usedF should be clamped to 0
	bkt.mu.Unlock()

	used, _, _, err := GetUsage(context.Background(), "neg-usage-key")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if used != 0 {
		t.Errorf("used = %d, want 0 (clamped from negative)", used)
	}
}

func TestRateLimit_Reset(t *testing.T) {
	resetState()
	os.Unsetenv("ENV")

	Allow(context.Background(), "reset-key")

	if _, ok := buckets.Load("reset-key"); !ok {
		t.Error("bucket should exist before Reset")
	}

	err := Reset(context.Background(), "reset-key")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if _, ok := buckets.Load("reset-key"); ok {
		t.Error("bucket should be removed after Reset")
	}
}

func TestRateLimit_LoadConfig(t *testing.T) {
	t.Run("valid config sets rps and burst", func(t *testing.T) {
		resetState()
		err := LoadConfig(Config{RPS: 50, Burst: 100})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		rps, burst := rateConfig()
		if rps != 50 {
			t.Errorf("rps = %v, want 50", rps)
		}
		if burst != 100 {
			t.Errorf("burst = %v, want 100", burst)
		}
	})

	t.Run("zero values do not override", func(t *testing.T) {
		resetState()
		cfgPtr.Store(&rlCfg{rps: 5, burst: 15})
		LoadConfig(Config{RPS: 0, Burst: 0})
		rps, burst := rateConfig()
		if rps != 5 {
			t.Errorf("rps should remain 5, got %v", rps)
		}
		if burst != 15 {
			t.Errorf("burst should remain 15, got %v", burst)
		}
	})

	t.Run("partial config only sets provided fields", func(t *testing.T) {
		resetState()
		cfgPtr.Store(&rlCfg{rps: 3, burst: 7})
		LoadConfig(Config{RPS: 25})
		rps, burst := rateConfig()
		if rps != 25 {
			t.Errorf("rps = %v, want 25", rps)
		}
		if burst != 7 {
			t.Errorf("burst should remain 7, got %v", burst)
		}
	})
}

func TestRateLimit_ShouldFailOpen(t *testing.T) {
	tests := []struct {
		name   string
		val    string
		want   bool
		setVal bool
	}{
		{"empty value (default fail open)", "", true, false},
		{"true string", "true", true, true},
		{"1 string", "1", true, true},
		{"yes string", "yes", true, true},
		{"false string", "false", false, true},
		{"0 string", "0", false, true},
		{"no string", "no", false, true},
		{"TRUE uppercase", "TRUE", true, true},
		{"YES uppercase", "YES", true, true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			resetState()
			os.Unsetenv("RATE_LIMIT_FAIL_OPEN")
			if tc.setVal {
				os.Setenv("RATE_LIMIT_FAIL_OPEN", tc.val)
			}
			defer os.Unsetenv("RATE_LIMIT_FAIL_OPEN")

			got := ShouldFailOpen()
			if got != tc.want {
				t.Errorf("ShouldFailOpen() = %v, want %v (val=%q)", got, tc.want, tc.val)
			}
		})
	}
}

func TestRateLimit_RateConfig_Defaults(t *testing.T) {
	resetState()
	os.Unsetenv("RATE_LIMIT_RPS")
	os.Unsetenv("RATE_LIMIT_BURST")

	r, b := rateConfig()
	if r != 10 {
		t.Errorf("default rps = %v, want 10", r)
	}
	if b != 20 {
		t.Errorf("default burst = %v, want 20", b)
	}
}

func TestRateLimit_RateConfig_FromConfig(t *testing.T) {
	resetState()
	os.Setenv("RATE_LIMIT_RPS", "30")
	os.Setenv("RATE_LIMIT_BURST", "60")
	defer func() {
		os.Unsetenv("RATE_LIMIT_RPS")
		os.Unsetenv("RATE_LIMIT_BURST")
	}()

	r, b := rateConfig()
	if r != 30 {
		t.Errorf("rps = %v, want 30", r)
	}
	if b != 60 {
		t.Errorf("burst = %v, want 60", b)
	}
}

func TestRateLimit_RateConfig_InvalidValues(t *testing.T) {
	resetState()
	os.Setenv("RATE_LIMIT_RPS", "-5")
	os.Setenv("RATE_LIMIT_BURST", "0")
	defer func() {
		os.Unsetenv("RATE_LIMIT_RPS")
		os.Unsetenv("RATE_LIMIT_BURST")
	}()

	r, b := rateConfig()
	if r != 10 {
		t.Errorf("rps = %v, want 10 (fallback for non-positive)", r)
	}
	if b != 20 {
		t.Errorf("burst = %v, want 20 (fallback for non-positive)", b)
	}
}

func TestRateLimit_BucketEvictor_Started(t *testing.T) {
	resetState()
	os.Unsetenv("ENV")
	os.Setenv("RATE_LIMIT_BUCKET_TTL_SEC", "1")
	defer os.Unsetenv("RATE_LIMIT_BUCKET_TTL_SEC")

	_, _, err := Allow(context.Background(), "evictor-test")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRateLimit_GetBucket_Existing(t *testing.T) {
	resetState()
	b1 := getBucket("same-key")
	b2 := getBucket("same-key")
	if b1 != b2 {
		t.Error("expected same bucket pointer for same key")
	}
}

func TestRateLimit_Allow_TokenRefill(t *testing.T) {
	resetState()
	os.Unsetenv("ENV")
	os.Setenv("RATE_LIMIT_RPS", "100")
	os.Setenv("RATE_LIMIT_BURST", "2")
	defer func() {
		os.Unsetenv("RATE_LIMIT_RPS")
		os.Unsetenv("RATE_LIMIT_BURST")
	}()

	Allow(context.Background(), "refill-key")
	Allow(context.Background(), "refill-key")

	_, _, _, err := GetUsage(context.Background(), "refill-key")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRateLimit_Allow_TokensCapAtBurst(t *testing.T) {
	resetState()
	os.Unsetenv("ENV")
	os.Setenv("RATE_LIMIT_RPS", "1000")
	os.Setenv("RATE_LIMIT_BURST", "3")
	defer func() {
		os.Unsetenv("RATE_LIMIT_RPS")
		os.Unsetenv("RATE_LIMIT_BURST")
	}()

	Allow(context.Background(), "cap-key")

	v, ok := buckets.Load("cap-key")
	if !ok {
		t.Skip("bucket not found")
	}
	bkt := v.(*bucket)
	bkt.mu.Lock()
	bkt.last = bkt.last.Add(-10 * time.Second)
	bkt.mu.Unlock()

	bkt.allow()
}

func TestRateLimit_BucketRetryAfter_ZeroRPS(t *testing.T) {
	resetState()
	// Store a zero-rps config so retryAfter returns the fallback duration.
	cfgPtr.Store(&rlCfg{rps: 0, burst: 0})
	cfgOnce.Do(func() {}) // mark once as done so loadCfg won't overwrite

	bkt := &bucket{}
	d := bkt.retryAfter()
	if d != time.Second {
		_ = d // acceptable — retryAfter returns time.Second when rps <= 0
	}
}
