package host

import "testing"

// ── Unit Tests ──────────────────────────────────────────────────────────

// Verifies DisableCoreDumps applies cleanly on the host running the suite: a no-op
// returning nil off-Linux, and a successful RLIMIT_CORE=0 set on Linux.
func TestDisableCoreDumps(t *testing.T) {
	if err := DisableCoreDumps(); err != nil {
		t.Fatalf("DisableCoreDumps() returned error: %v", err)
	}
}
