//go:build linux

package host

import "syscall"

// Sets the core-dump size limit to zero so a crash cannot spill in-memory secrets — notably an
// RSA signing key — to disk. Applies on the Linux container that ships; non-Linux dev hosts get
// the no-op in hardening_other.go.
func DisableCoreDumps() error {
	return syscall.Setrlimit(syscall.RLIMIT_CORE, &syscall.Rlimit{Cur: 0, Max: 0})
}
