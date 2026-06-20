//go:build linux

package host

import "golang.org/x/sys/unix"

func DisableCoreDumps() error {
	return unix.Setrlimit(unix.RLIMIT_CORE, &unix.Rlimit{Cur: 0, Max: 0})
}

func DisableTracing() error {
	return unix.Prctl(unix.PR_SET_DUMPABLE, 0, 0, 0, 0)
}

func LockMemory() error {
	return unix.Mlockall(unix.MCL_CURRENT | unix.MCL_FUTURE)
}
