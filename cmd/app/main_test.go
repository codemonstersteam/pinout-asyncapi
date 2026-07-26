package main

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// version — стабильная команда, exit 0 + непустой вывод. Дублирует контракт smoke.
func TestVersionCmd(t *testing.T) {
	root := newRootCmd()
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs([]string{"version"})
	if err := root.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out.String(), "version") {
		t.Fatalf("ожидали строку версии, получили %q", out.String())
	}
}

// validate <config.yaml> — the live wiring's egress adapter must surface CONFIG_ERROR (exit 2)
// for a config path that doesn't exist (ConfigStore.Load's failure branch), with no report
// printed to stdout (use-case.md Minimal guarantee) and the error logged to stderr. This is a
// thin structural check on the wiring (ticket-23); the full pipe's business-logic scenarios are
// proven by internal/validate's own unit tests and by the component-test suite, not here.
func TestValidateCmd_ConfigFileNotFound(t *testing.T) {
	root := newRootCmd()
	var out, errb bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&errb)
	root.SetArgs([]string{"validate", "does-not-exist.yaml"})
	err := root.Execute()
	if err == nil {
		t.Fatal("ожидали exitError, получили nil")
	}
	var ee *exitError
	if !errors.As(err, &ee) || ee.code != 2 {
		t.Fatalf("ожидали exit code 2, получили %v", err)
	}
	if strings.TrimSpace(out.String()) != "" {
		t.Fatalf("ожидали пустой stdout на CONFIG_ERROR, получили %q", out.String())
	}
	if errb.Len() == 0 {
		t.Fatal("ожидали диагностику в stderr, получили пусто")
	}
}

// validate <config.yaml> — cli.ResolveExitCode's `default:` arm (C4, the out-of-taxonomy
// fallback → exit 3, contracts.md §5, D3). ReportWriter.Write fails at
// internal/validate/io_report.go:53 (os.WriteFile against a non-existent directory) with an
// error that wraps NONE of errors.go's closed sentinels, so ProcessValidate's returned error
// falls through every named case in ResolveExitCode's switch into its default arm. Ticket-03
// (docs/design/slice-01-validate/changes/001-pipe-arity-coverage-gaps/tickets/ticket-03.md):
// MUST assert nothing about the stdout body here — on this path it is known schema-invalid
// (code:"" , no subject), recorded as debt (change-delta.md §3 row B), not repaired by this
// ticket.
func TestValidateCmd_ReportWriteFailure_FallsBackToExitThree(t *testing.T) {
	dir := t.TempDir()

	// Reachable, compatible pair (the same fixtures scenario 1 / the determinism anchor use),
	// referenced directly — resolved relative to this test file's package directory (go test's
	// cwd), not copied.
	consumedContractPath, err := filepath.Abs("../../component-tests/fixtures/validate/good/consumed-contract.yaml")
	if err != nil {
		t.Fatalf("abs consumed-contract path: %v", err)
	}
	providerSpecPath, err := filepath.Abs("../../component-tests/fixtures/validate/good/provider-asyncapi.yaml")
	if err != nil {
		t.Fatalf("abs provider-asyncapi path: %v", err)
	}

	// json_report_file inside a non-existent directory — os.WriteFile fails with ENOENT, an
	// error that wraps no sentinel from errors.go.
	reportPath := filepath.Join(dir, "no-such-subdir", "report.json")

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
  save_json_report: true
  json_report_file: %s
  timeout: 5
`, consumedContractPath, providerSpecPath, reportPath)
	if err := os.WriteFile(configPath, []byte(configYAML), 0o644); err != nil {
		t.Fatalf("write config.yaml: %v", err)
	}

	root := newRootCmd()
	var out, errb bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&errb)
	root.SetArgs([]string{"validate", configPath})
	execErr := root.Execute()

	var ee *exitError
	if !errors.As(execErr, &ee) || ee.code != 3 {
		t.Fatalf("ожидали exit code 3 (out-of-taxonomy fallback), получили %v", execErr)
	}
}

// validate with argc ≠ 1 (C1) — cobra's own Args validator (cobra.ExactArgs(1), newValidateCmd)
// rejects the invocation before cli.Parse and the pipe ever run, so cli.ResolveExitCode is never
// reached: the returned error must NOT be an *exitError. Two cases: 0 args and 2 args.
func TestValidateCmd_ArgcZero_NeverReachesExitCode(t *testing.T) {
	root := newRootCmd()
	var out, errb bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&errb)
	root.SetArgs([]string{"validate"})
	err := root.Execute()

	if err == nil {
		t.Fatal("ожидали ошибку от cobra.ExactArgs(1), получили nil")
	}
	var ee *exitError
	if errors.As(err, &ee) {
		t.Fatalf("ожидали НЕ *exitError (argc-guard short-circuit), получили exitError code=%d", ee.code)
	}
}

func TestValidateCmd_ArgcTwo_NeverReachesExitCode(t *testing.T) {
	root := newRootCmd()
	var out, errb bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&errb)
	root.SetArgs([]string{"validate", "a", "b"})
	err := root.Execute()

	if err == nil {
		t.Fatal("ожидали ошибку от cobra.ExactArgs(1), получили nil")
	}
	var ee *exitError
	if errors.As(err, &ee) {
		t.Fatalf("ожидали НЕ *exitError (argc-guard short-circuit), получили exitError code=%d", ee.code)
	}
}
