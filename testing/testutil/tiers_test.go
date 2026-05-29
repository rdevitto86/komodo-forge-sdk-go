package testutil

import "testing"

func TestResolve(t *testing.T) {
	tests := []struct {
		name  string
		short bool
		env   string
		want  tier
	}{
		{"short overrides tier", true, "chaos", unit},
		{"default is component", false, "", component},
		{"unrecognized falls back to component", false, "bogus", component},
		{"explicit unit", false, "unit", unit},
		{"explicit component", false, "component", component},
		{"explicit integration", false, "integration", integration},
		{"explicit e2e", false, "e2e", e2e},
		{"explicit chaos", false, "chaos", chaos},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := resolve(tt.short, tt.env); got != tt.want {
				t.Errorf("resolve(%t, %q) = %d, want %d", tt.short, tt.env, got, tt.want)
			}
		})
	}
}

func TestActiveTierGating(t *testing.T) {
	tests := []struct {
		name        string
		short       bool
		env         string
		integration bool
		e2e         bool
		chaos       bool
	}{
		{"short forces unit-only", true, "chaos", false, false, false},
		{"default skips above component", false, "", false, false, false},
		{"integration runs integration only", false, "integration", true, false, false},
		{"e2e runs through e2e", false, "e2e", true, true, false},
		{"chaos runs all", false, "chaos", true, true, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := resolve(tt.short, tt.env)
			if (got >= integration) != tt.integration {
				t.Errorf("integration enabled = %t, want %t", got >= integration, tt.integration)
			}
			if (got >= e2e) != tt.e2e {
				t.Errorf("e2e enabled = %t, want %t", got >= e2e, tt.e2e)
			}
			if (got >= chaos) != tt.chaos {
				t.Errorf("chaos enabled = %t, want %t", got >= chaos, tt.chaos)
			}
		})
	}
}
