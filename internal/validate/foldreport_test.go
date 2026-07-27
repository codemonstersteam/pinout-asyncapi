package validate

import (
	"reflect"
	"testing"
	"time"
)

// fixedClock is a fake Clock (contracts.md §FoldReport) returning a fixed time.Time, so
// FoldReport's generated_at is deterministic in tests without touching the system clock.
type fixedClock struct {
	now time.Time
}

func (c fixedClock) Now() time.Time { return c.now }

// baseFoldReportOutcome is the shared Outcome fixture both FoldReport tests start from — a valid
// Outcome (CompareContracts's own consequent), varied only by Violations per the compatible <=>
// errors == [] formula (contracts.md §4).
func baseFoldReportOutcome() Outcome {
	return Outcome{
		UncoveredChannels: []string{"WALLET.NOTIFY.EVENT"},
		ConsumerName:      "wallet-consumer",
		Provenance: Provenance{
			Provider:        "wallet-provider",
			ProviderVersion: "1.0.0",
			CapturedHash:    "sha256:0000000000000000000000000000000000000000000000000000000000000000",
		},
	}
}

// TestFoldReport_Compatible is the happy path: an Outcome with no Violations folds into a Report
// with compatible == true and errors == [] — clock.Now() supplies generated_at, called exactly
// once (D10).
func TestFoldReport_Compatible(t *testing.T) {
	clock := fixedClock{now: time.Date(2026, 7, 25, 12, 0, 0, 0, time.UTC)}
	outcome := baseFoldReportOutcome()

	got := BuildReporter(clock).Fold(outcome)

	want := Report{
		SchemaVersion:     "1.1",
		Validator:         "pinout-asyncapi",
		Interaction:       "async",
		Consumer:          ReportConsumer{Name: "wallet-consumer"},
		GeneratedAt:       "2026-07-25T12:00:00Z",
		Compatible:        true,
		Provenance:        outcome.Provenance,
		Errors:            []Violation{},
		UncoveredChannels: []string{"WALLET.NOTIFY.EVENT"},
	}

	if !reflect.DeepEqual(got, want) {
		t.Errorf("FoldReport() = %+v, want %+v", got, want)
	}
}

// TestFoldReport_Incompatible is the branch: an Outcome carrying at least one Violation folds into
// a Report with compatible == false and errors[] carrying the Violations through unchanged —
// FoldReport never recomputes subject/location (contracts.md §2), just carries Violation -> errors[].
func TestFoldReport_Incompatible(t *testing.T) {
	clock := fixedClock{now: time.Date(2026, 7, 25, 12, 0, 0, 0, time.UTC)}
	outcome := baseFoldReportOutcome()
	outcome.Violations = []Violation{
		{
			Code:     CodeChannelNotInProvider,
			Message:  "channel not declared by the provider",
			Subject:  "WALLET.BALANCE.REQUEST send",
			Location: "WALLET.BALANCE.REQUEST send",
		},
	}

	got := BuildReporter(clock).Fold(outcome)

	if got.Compatible {
		t.Errorf("FoldReport() Compatible = true, want false when errors != []")
	}
	if !reflect.DeepEqual(got.Errors, outcome.Violations) {
		t.Errorf("FoldReport() Errors = %+v, want Violations carried through unchanged: %+v", got.Errors, outcome.Violations)
	}
}
