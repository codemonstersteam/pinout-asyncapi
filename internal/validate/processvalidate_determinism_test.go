package validate

import (
	"bytes"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

	"pinout-asyncapi/internal/shared/report"
)

// TestProcessValidateDeterminism is a D10 DETERMINISM REGRESSION ANCHOR, not a head unit test —
// the same documented, non-formula status internal/validate/report_json_shape_test.go already
// carries (contracts.md §ProcessValidate). It adds no row to contracts.md §4; the head stays not
// unit-tested (C0/Q2 stand verbatim).
//
// Captured from the PRE-REFACTOR pipe (change-delta.md §3 row E, module-tree.md §4 closing note):
// FoldReport is today the only module reading the clock, and it is documented to call Now()
// exactly once per Fold. This test proves that invariant end-to-end through ProcessValidate with a
// COUNTING fake Clock, and pins the full byte shape of the resulting Report (via the same
// report.WriteJSON the binary uses for stdout) against testdata/report.golden.json — including the
// real generated_at value this frozen clock produces, since the clock is frozen the bytes are
// naturally deterministic and reproducible.
//
// MUST NOT be regenerated after ticket-04/05/06 move the clock read into Reporter.Fold — doing so
// would void the proof this anchor exists to provide (it must stay green both before and after,
// since ProcessValidate's own signature is unchanged by that reshape).
func TestProcessValidateDeterminism(t *testing.T) {
	dir := t.TempDir()

	// Reads the SAME bytes as component-tests/fixtures/validate/good/{consumed-contract,
	// provider-asyncapi}.yaml — never fork them, so this anchor and ticket-01's I2 baseline cannot
	// drift apart. Paths are relative to this test file's package directory (go test's cwd).
	consumedContractPath := copyFixture(t, dir, "../../component-tests/fixtures/validate/good/consumed-contract.yaml", "consumed-contract.yaml")
	providerSpecPath := copyFixture(t, dir, "../../component-tests/fixtures/validate/good/provider-asyncapi.yaml", "provider-asyncapi.yaml")

	configPath := filepath.Join(dir, "config.yaml")
	configYAML := fmt.Sprintf(`consumer:
  name: mq-rest-sync-adapter
  consumed_contract_path: %s
  channels:
    - WALLET.BALANCE.REQUEST
    - WALLET.BALANCE.RESPONSE
    - WALLET.AUDIT
    - WALLET.LEDGER.EVENTS
provider:
  name: wallet-balance
  spec_path: %s
settings:
  save_json_report: false
  timeout: 5
`, consumedContractPath, providerSpecPath)
	if err := os.WriteFile(configPath, []byte(configYAML), 0o644); err != nil {
		t.Fatalf("write config.yaml: %v", err)
	}

	inv := Invocation{ConfigPath: configPath}
	clock := &countingClock{now: time.Date(2026, 7, 25, 12, 0, 0, 0, time.UTC)}
	deps := Deps{Clock: clock, HTTPClient: &http.Client{}}

	got, err := ProcessValidate(inv, deps)
	if err != nil {
		t.Fatalf("ProcessValidate() error = %v, want nil", err)
	}

	// Assertion 1 (I1a): exactly one Now() call across the whole ProcessValidate run. A Reporter
	// reading the clock twice, or per-violation, fails here — foldreport_test.go's existing
	// fixedClock cannot see this (it returns the same instant every time and counts nothing).
	if clock.calls != 1 {
		t.Errorf("Clock.Now() called %d times, want exactly 1 (D10)", clock.calls)
	}

	// Assertion 2 (I1c): full-byte comparison against the golden, marshaled with the same writer
	// the binary uses for stdout. No normalization, no struct comparison — generated_at is
	// deterministic here because the clock is frozen.
	var buf bytes.Buffer
	if err := report.WriteJSON(&buf, got); err != nil {
		t.Fatalf("report.WriteJSON(got): %v", err)
	}

	goldenPath := filepath.Join("testdata", "report.golden.json")
	want, err := os.ReadFile(goldenPath)
	if err != nil {
		t.Fatalf("read golden %s: %v", goldenPath, err)
	}

	if !bytes.Equal(buf.Bytes(), want) {
		t.Errorf("ProcessValidate() report bytes mismatch against %s\ngot:\n%s\nwant:\n%s", goldenPath, buf.String(), string(want))
	}
}

// countingClock is a fake Clock (domain.go's Clock port) that returns a fixed instant on every
// call but — unlike foldreport_test.go's fixedClock — COUNTS how many times Now() was invoked, so
// this test can assert the D10 "exactly one clock read" invariant that a same-instant-every-time
// fake cannot see.
type countingClock struct {
	now   time.Time
	calls int
}

func (c *countingClock) Now() time.Time {
	c.calls++
	return c.now
}

// copyFixture reads srcRelPath (relative to this package's directory) and writes its bytes
// unchanged into destDir/destName, returning the new absolute path. Used so the determinism
// anchor's config points at a filesystem-real path while reading the exact same consumed-contract
// / provider-spec bytes component-tests/fixtures/validate/good/ already carries.
func copyFixture(t *testing.T, destDir, srcRelPath, destName string) string {
	t.Helper()

	data, err := os.ReadFile(srcRelPath)
	if err != nil {
		t.Fatalf("read fixture %s: %v", srcRelPath, err)
	}

	destPath := filepath.Join(destDir, destName)
	if err := os.WriteFile(destPath, data, 0o644); err != nil {
		t.Fatalf("write fixture copy %s: %v", destPath, err)
	}

	return destPath
}
