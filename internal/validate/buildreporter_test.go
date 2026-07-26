package validate

import (
	"testing"
	"time"
)

// TestBuildReporter realizes contracts.md §4's formula for BuildReporter: N=1, happy only — the
// factory binds the clock port and has no antecedent branch (contracts.md §BuildReporter,
// ADR-001 node 18).
func TestBuildReporter(t *testing.T) {
	clock := fixedClock{now: time.Date(2026, 7, 25, 12, 0, 0, 0, time.UTC)}

	reporter := BuildReporter(clock)

	if reporter.clock != clock {
		t.Fatalf("BuildReporter().clock = %v, want %v", reporter.clock, clock)
	}
}
