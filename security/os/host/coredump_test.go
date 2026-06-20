package host

import "testing"

func TestDisableCoreDumps(t *testing.T) {
	if err := DisableCoreDumps(); err != nil {
		t.Fatalf("DisableCoreDumps() returned error: %v", err)
	}
}

func TestDisableTracing(t *testing.T) {
	if err := DisableTracing(); err != nil {
		t.Fatalf("DisableTracing() returned error: %v", err)
	}
}

func TestLockMemory(t *testing.T) {
	_ = LockMemory()
}
