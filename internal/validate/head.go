// head.go holds ProcessValidate, the slice's composition root (module-tree.md §6, §4 head-pipe
// pseudocode; contracts.md §ProcessValidate) — a linear ROP pipe wiring already-implemented,
// already-tested parts in sequence. No logic of its own; not unit-tested (module-tree.md §3 Step
// 8.1 hard rule — head/orchestration modules are proven by component tests, ticket-23).
package validate

import (
	"net/http"
)

// Deps is ProcessValidate's ports bundle (contracts.md §ProcessValidate "Dependencies"):
// declared HERE, not register.go, per ticket-22's composition-root convention (the head owns the
// ports it needs; register.go — ticket-23 — wires the concrete instance into cmd/pinout-asyncapi,
// which would otherwise forward-reference this file's type). ConfigStore/ContractStore need no
// port — both are zero-field, constructed directly in the pipe below (contracts.md
// §ConfigStore.Load/§ContractStore.Load, "Dependencies: —"); only the two orthogonal tools that
// cross an I/O/testability boundary are injected: Clock (ticket-03, D10 — the core never reads
// system time directly) and *http.Client (passed through, unopened, to BuildSpecLoader, ticket
// 13 — never exposed to the head itself). No raw *os.File.
type Deps struct {
	Clock      Clock
	HTTPClient *http.Client
}

// ProcessValidate is the slice's head (contracts.md §ProcessValidate, module-tree.md §4): the
// linear ROP pipe from a well-formed Invocation to a canon-1.1 Report, short-circuiting on the
// FIRST step's Error, risen untransformed (railway-oriented programming — module-tree.md §4).
// BuildSpecLoader runs late — after NewConfig — so cfg.Settings.Timeout is always the real,
// already-validated value (EMULATION.md S15; this is what makes the TIMEOUT_ERROR scenario
// reachable). CompareContracts never returns an error on the domain axis — incompatible is a
// legitimate Outcome value, not an Err (use-case.md, note after Extensions 6a-6i).
//
// Antecedent: a well-formed Invocation.
// Consequent: success -> Report; failure -> the first short-circuiting step's Error, untransformed.
func ProcessValidate(inv Invocation, deps Deps) (Report, error) {
	rawConfig, err := (ConfigStore{}).Load(inv.ConfigPath)
	if err != nil {
		return Report{}, err
	}

	cfg, err := NewConfig(rawConfig)
	if err != nil {
		return Report{}, err
	}

	rawContract, err := (ContractStore{}).Load(cfg.consumer.ConsumedContractPath)
	if err != nil {
		return Report{}, err
	}

	contract, err := NewConsumedContract(rawContract, cfg.consumer.Name)
	if err != nil {
		return Report{}, err
	}

	// Late-bound factory call (BuildSpecLoader is pure/total — no Result, no short-circuit here):
	// must run after NewConfig so cfg.settings.Timeout is the real, validated value.
	loader := BuildSpecLoader(cfg.provider, cfg.settings.Timeout, deps.HTTPClient)

	spec, err := loader.Load()
	if err != nil {
		return Report{}, err
	}

	pchannels := DeriveProviderChannels(spec)

	comparison, err := NewComparison(cfg, contract, pchannels)
	if err != nil {
		return Report{}, err
	}

	outcome := CompareContracts(comparison)

	report := FoldReport(outcome, deps.Clock)

	writer := BuildReportWriter(cfg.settings)

	report, err = writer.Write(report)
	if err != nil {
		return Report{}, err
	}

	return report, nil
}
