package steps

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/cucumber/godog"
)

// fixturesDir — per-slice fixture root, bind-mounted read-only into the tester container
// (docker-compose.test.yml: ./fixtures:/fixtures:ro). Isolated per slice: only
// component-tests/fixtures/validate/**.
const fixturesDir = "/fixtures/validate"

// validateReport — the wire shape of api-specification/report.schema.json, just enough of it
// for black-box assertions (exit code + stdout JSON). Mirrors the frozen contract; not a copy
// of any internal type (component tests know nothing of internals).
type validateReport struct {
	SchemaVersion string `json:"schema_version"`
	Compatible    *bool  `json:"compatible"`
	Provenance    struct {
		Provider        string `json:"provider"`
		ProviderVersion string `json:"provider_version"`
		CapturedHash    string `json:"captured_hash"`
	} `json:"provenance"`
	Errors []struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"errors"`
	UncoveredChannels []string `json:"uncovered_channels"`
}

// registerValidateSteps — step-defs for slice-01-validate's component scenarios
// (component-tests/features/validate.feature), realized 1:1 from contracts.md §6
// "Component-scenario set". Mechanical glue only (subprocess + JSON-of-stdout assertions);
// no domain logic here — the black box is the staged binary, not this package.
func (w *World) registerValidateSteps(ctx *godog.ScenarioContext) {
	// Given — select the per-scenario fixture (good / bad-<branch>).
	ctx.Step(`^a config whose consumed-contract and provider spec are reachable and compatible$`, w.givenGoodConfig)
	ctx.Step(`^a config where the provider spec source is not exactly one of spec_path/spec_url$`, w.givenConfigErrorFixture)
	ctx.Step(`^a config pointing at a consumed-contract path that does not exist$`, w.givenContractFileNotFoundFixture)
	ctx.Step(`^a config pointing at a consumed-contract whose consumer does not match config\.consumer\.name$`, w.givenContractParseErrorFixture)
	ctx.Step(`^a config pointing at a provider spec_path that does not exist$`, w.givenSpecFileNotFoundFixture)
	ctx.Step(`^a config pointing at a provider spec that is not valid AsyncAPI$`, w.givenSpecParseErrorFixture)
	ctx.Step(`^a config with spec_url pointing at a stub that returns 404$`, w.givenHTTPErrorFixture)
	ctx.Step(`^a config with spec_url pointing at a stub that stalls past settings\.timeout$`, w.givenTimeoutFixture)
	ctx.Step(`^a config whose consumer uses a channel address absent from the provider spec$`, w.givenIncompatibleFixture)

	// When — run the staged binary against the selected config fixture.
	ctx.Step("^I run `pinout-asyncapi validate config\\.yaml`$", w.runValidate)

	// Then — exit code (English phrasing alongside cli_steps.go's Russian one; same method).
	ctx.Step(`^the exit code is (\d+)$`, w.exitCode)

	// Then — stdout report assertions.
	ctx.Step(`^stdout is a schema-valid report with compatible=true and errors==\[\]$`, w.stdoutCompatibleNoErrors)
	ctx.Step(`^uncovered provider channels are listed in uncovered_channels\[\]$`, w.stdoutHasUncoveredChannels)
	ctx.Step(`^no report is printed on stdout$`, w.stdoutHasNoReport)
	ctx.Step(`^stdout report errors\[0\]\.code == "([^"]*)"$`, w.stdoutFirstErrorCode)
	ctx.Step(`^stdout is a schema-valid report with compatible=false$`, w.stdoutIncompatible)
	ctx.Step(`^stdout bytes equal the captured baseline report$`, w.stdoutBytesEqualBaseline)
}

func (w *World) givenGoodConfig()         { w.configPath = fixturesDir + "/good/config.yaml" }
func (w *World) givenConfigErrorFixture() { w.configPath = fixturesDir + "/bad-config/config.yaml" }
func (w *World) givenContractFileNotFoundFixture() {
	w.configPath = fixturesDir + "/bad-file-not-found-contract/config.yaml"
}
func (w *World) givenContractParseErrorFixture() {
	w.configPath = fixturesDir + "/bad-parse-error-contract/config.yaml"
}
func (w *World) givenSpecFileNotFoundFixture() {
	w.configPath = fixturesDir + "/bad-file-not-found-spec/config.yaml"
}
func (w *World) givenSpecParseErrorFixture() {
	w.configPath = fixturesDir + "/bad-parse-error-spec/config.yaml"
}
func (w *World) givenHTTPErrorFixture() { w.configPath = fixturesDir + "/bad-http-error/config.yaml" }
func (w *World) givenTimeoutFixture()   { w.configPath = fixturesDir + "/bad-timeout/config.yaml" }

// givenIncompatibleFixture — scenario 9 (contracts.md §6 row 9): good/ + one channel address
// added to both consumer.channels and the consumed-contract's channels, absent from the
// provider spec. Triggers R1 (CHANNEL_NOT_IN_PROVIDER) -> exit 1, the primary incompatible
// verdict — a happy-class member (CompareContracts returns it as a value, never an Error), not
// an 8th adapter branch.
func (w *World) givenIncompatibleFixture() {
	w.configPath = fixturesDir + "/incompatible/config.yaml"
}

// runValidate invokes the staged binary against the selected config fixture — the black-box
// equivalent of the Cockburn `pinout-asyncapi validate <config.yaml>` invocation. ticket-23
// renamed the command surface from the scaffold's placeholder `run` to `validate`
// (contracts.md, module-tree.md §4); this exec call matches that rename.
func (w *World) runValidate() error {
	if w.configPath == "" {
		return fmt.Errorf("no config fixture selected — a Given step must run first")
	}
	return w.exec([]string{"validate", w.configPath})
}

// parseReport decodes stdout+stderr (CombinedOutput, per exec in cli_steps.go) as the report
// JSON. A decode failure is a legitimate scenario failure — RED by business reason while the
// slice is unimplemented (placeholder prints a NOT_IMPLEMENTED body, not a canon-1.1 report),
// not a harness bug.
func (w *World) parseReport() (validateReport, error) {
	var rep validateReport
	if err := json.Unmarshal([]byte(w.lastOut), &rep); err != nil {
		return rep, fmt.Errorf("stdout is not a valid JSON report (%v); raw output: %s", err, w.lastOut)
	}
	return rep, nil
}

func (w *World) stdoutCompatibleNoErrors() error {
	rep, err := w.parseReport()
	if err != nil {
		return err
	}
	if rep.Compatible == nil || !*rep.Compatible {
		return fmt.Errorf("expected report.compatible=true, got %+v (errors=%+v)", rep.Compatible, rep.Errors)
	}
	if len(rep.Errors) != 0 {
		return fmt.Errorf("expected report.errors==[], got %+v", rep.Errors)
	}
	return nil
}

func (w *World) stdoutHasUncoveredChannels() error {
	rep, err := w.parseReport()
	if err != nil {
		return err
	}
	if len(rep.UncoveredChannels) == 0 {
		return fmt.Errorf("expected a non-empty report.uncovered_channels[], got %+v", rep.UncoveredChannels)
	}
	return nil
}

// stdoutHasNoReport asserts CONFIG_ERROR's contract (contracts.md §5): no report is written
// before a CONFIG_ERROR short-circuit. Empty stdout trivially satisfies this; otherwise stdout
// must NOT decode as a canon-1.1 report body (schema_version=="1.1" + compatible present).
func (w *World) stdoutHasNoReport() error {
	if strings.TrimSpace(w.lastOut) == "" {
		return nil
	}
	var rep validateReport
	if err := json.Unmarshal([]byte(w.lastOut), &rep); err == nil && rep.SchemaVersion == "1.1" && rep.Compatible != nil {
		return fmt.Errorf("expected no report on stdout for CONFIG_ERROR, got a schema_version=1.1 report body: %s", w.lastOut)
	}
	return nil
}

func (w *World) stdoutFirstErrorCode(wantCode string) error {
	rep, err := w.parseReport()
	if err != nil {
		return err
	}
	if len(rep.Errors) == 0 {
		return fmt.Errorf("expected report.errors[0].code=%s, got empty errors[]", wantCode)
	}
	if got := rep.Errors[0].Code; got != wantCode {
		return fmt.Errorf("expected report.errors[0].code=%s, got %s", wantCode, got)
	}
	return nil
}

// stdoutIncompatible asserts scenario 9's verdict half (contracts.md §6, assertion order):
// compatible == false. The canon-1.1 invariant compatible <=> errors == [] means a false
// verdict must carry at least one error; the specific errors[0].code is its own Then-step.
func (w *World) stdoutIncompatible() error {
	rep, err := w.parseReport()
	if err != nil {
		return err
	}
	if rep.Compatible == nil || *rep.Compatible {
		return fmt.Errorf("expected report.compatible=false, got %+v", rep.Compatible)
	}
	if len(rep.Errors) == 0 {
		return fmt.Errorf("expected a non-empty report.errors[] alongside compatible=false (canon-1.1 invariant compatible<=>errors==[]), got empty")
	}
	return nil
}

// generatedAtField — matches the single "generated_at": "<value>" occurrence in a canon-1.1
// report body. Used only to normalize the ONE surgical byte difference the I2 baseline compare
// (contracts.md §5, new I2 row) allows: generated_at's value, never any other byte.
var generatedAtField = regexp.MustCompile(`"generated_at"\s*:\s*"([^"]*)"`)

// normalizeGeneratedAt replaces generated_at's value with a fixed RFC3339 placeholder, after
// confirming the original value is present and itself parses as RFC3339 — the report's clock
// port (D10) is proven working, just not pinned byte-for-byte. Every other byte is untouched.
func normalizeGeneratedAt(b []byte) ([]byte, error) {
	loc := generatedAtField.FindSubmatchIndex(b)
	if loc == nil {
		return nil, fmt.Errorf("generated_at field not found in report bytes: %s", b)
	}
	val := string(b[loc[2]:loc[3]])
	if _, err := time.Parse(time.RFC3339, val); err != nil {
		return nil, fmt.Errorf("generated_at %q does not parse as RFC3339: %w", val, err)
	}
	out := make([]byte, 0, len(b))
	out = append(out, b[:loc[2]]...)
	out = append(out, []byte("1970-01-01T00:00:00Z")...)
	out = append(out, b[loc[3]:]...)
	return out, nil
}

// stdoutBytesEqualBaseline — the I2 byte baseline (contracts.md §5, new I2 row; change-delta.md
// §3 row D). Full byte equality between stdout and the captured
// fixtures/validate/good/report.baseline.json, with exactly one surgical normalization
// (generated_at's value on both sides). Deliberately does NOT decode-and-compare structs:
// indentation, key order, and every constant must match, which a struct compare would hide.
func (w *World) stdoutBytesEqualBaseline() error {
	baselinePath := fixturesDir + "/good/report.baseline.json"
	baselineBytes, err := os.ReadFile(baselinePath)
	if err != nil {
		return fmt.Errorf("reading baseline %s: %w", baselinePath, err)
	}
	normActual, err := normalizeGeneratedAt([]byte(w.lastOut))
	if err != nil {
		return fmt.Errorf("actual stdout: %w", err)
	}
	normBaseline, err := normalizeGeneratedAt(baselineBytes)
	if err != nil {
		return fmt.Errorf("baseline file %s: %w", baselinePath, err)
	}
	if !bytes.Equal(normActual, normBaseline) {
		return fmt.Errorf("stdout bytes differ from baseline after generated_at normalization:\n--- actual ---\n%s\n--- baseline ---\n%s", normActual, normBaseline)
	}
	return nil
}
