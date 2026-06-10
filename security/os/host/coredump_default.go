//go:build !linux

// Package host provides OS-level security primitives for services that hold secrets
// in memory (signing keys, cache credentials). The guards are build-tagged: they apply
// on the Linux container that ships and degrade to no-ops on non-Linux dev hosts,
// so callers can invoke them unconditionally at startup.
package host

// Returns nil on non-Linux dev hosts (macOS/Windows). Core-dump hardening is a Linux-container
// concern; the shipping artifact is always a Linux image regardless of build/deploy host.
func DisableCoreDumps() error { return nil }
