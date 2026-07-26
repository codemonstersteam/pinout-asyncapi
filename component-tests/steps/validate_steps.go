package steps

import (
	"encoding/json"
	"fmt"
	"strings"

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

	// When — run the staged binary against the selected config fixture.
	ctx.Step("^I run `pinout-asyncapi validate config\\.yaml`$", w.runValidate)

	// Then — exit code (English phrasing alongside cli_steps.go's Russian one; same method).
	ctx.Step(`^the exit code is (\d+)$`, w.exitCode)

	// Then — stdout report assertions.
	ctx.Step(`^stdout is a schema-valid report with compatible=true and errors==\[\]$`, w.stdoutCompatibleNoErrors)
	ctx.Step(`^uncovered provider channels are listed in uncovered_channels\[\]$`, w.stdoutHasUncoveredChannels)
	ctx.Step(`^no report is printed on stdout$`, w.stdoutHasNoReport)
	ctx.Step(`^stdout report errors\[0\]\.code == "([^"]*)"$`, w.stdoutFirstErrorCode)
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
