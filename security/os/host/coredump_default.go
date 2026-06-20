//go:build !linux

package host

func DisableCoreDumps() error { return nil }

func DisableTracing() error { return nil }

func LockMemory() error { return nil }
