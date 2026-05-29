package testutil

import (
	"os"
	"testing"
)

type tier int

const (
	unit tier = iota
	component
	integration
	e2e
	chaos
)

func active() tier {
	return resolve(testing.Short(), os.Getenv("TEST_TIER"))
}

func resolve(short bool, env string) tier {
	if short {
		return unit // -short overrides TEST_TIER and forces unit-only
	}
	switch env {
	case "unit":
		return unit
	case "component":
		return component
	case "integration":
		return integration
	case "e2e":
		return e2e
	case "chaos":
		return chaos
	default:
		return component
	}
}

func require(t *testing.T, want tier, label string) {
	t.Helper()
	if active() < want {
		t.Skipf("skipping %s test: set TEST_TIER=%s or higher to run", label, label)
	}
}

func Integration(t *testing.T) {
	t.Helper()
	require(t, integration, "integration")
}

func E2E(t *testing.T) {
	t.Helper()
	require(t, e2e, "e2e")
}

func Chaos(t *testing.T) {
	t.Helper()
	require(t, chaos, "chaos")
}
