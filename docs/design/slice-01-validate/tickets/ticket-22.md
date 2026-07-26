---
id: 22
type: module
slice: slice-01-validate
blocked_by: [01, 02, 03, 06, 07, 08, 09, 11, 13, 14, 15, 18, 19, 20, 21]
inputs: [docs/design/slice-01-validate/module-tree.md, docs/design/slice-01-validate/contracts.md]
outputs: [internal/validate/head.go]
io: none
skills: []
---

### TICKET 22 — slice-01-validate/ProcessValidate (head): the ROP composition pipe

**io:** none → skills: [] (composition root — a linear ROP pipe of already-tested parts; no
branching of its own, **not unit-tested**).

**Context (only this module):**
- **Declare the `Deps` ports struct HERE, in `head.go`** (composition-root convention — the head
  owns the ports it needs; `module-tree.md` §6 assigns `Deps{...}` to `register.go`, but that row
  describes wiring a *concrete instance*, not the type declaration — declaring the type here
  avoids a forward reference from this ticket to the wiring ticket, and matches the sync
  sibling's convention). `Deps{ ConfigStore ConfigStore, ContractStore ContractStore, Timeout
  time.Duration, HTTPClient *http.Client, ReportSettings func(Config) Settings, Clock Clock }` —
  in practice: autonomous I/O objects (`ConfigStore`, ticket 06; `ContractStore`, ticket 08) +
  the orthogonal tools `Clock` (ticket 03) and `*http.Client` (passed through to
  `BuildSpecLoader`, ticket 13) — **no** raw `*os.File`.
- contract (`contracts.md` §ProcessValidate + `module-tree.md` §4 head-pipe pseudocode):
  `ProcessValidate(inv: Invocation, d: Deps) -> Result[Report, Error]`. Linear pipe, short-circuit
  on the first sentinel, risen **untransformed**:
  ```
  ConfigStore.Load(inv.ConfigPath)                     -> RawConfig
  NewConfig(raw)                                        -> Config
  ContractStore.Load(cfg.Consumer.ConsumedContractPath) -> RawContract
  NewConsumedContract(raw, cfg.Consumer.Name)           -> ConsumedContract
  loader := BuildSpecLoader(cfg.Provider, cfg.Settings.Timeout, d.HTTPClient) -> SpecLoader   # LATE — after NewConfig
  loader.Load()                                         -> ProviderSpec
  DeriveProviderChannels(spec)                          -> ProviderChannels
  NewComparison(cfg, contract, pchans)                  -> Comparison
  CompareContracts(comparison)                          -> Outcome
  FoldReport(outcome, d.Clock)                          -> Report
  writer := BuildReportWriter(cfg.Settings)             -> ReportWriter
  writer.Write(report)                                  -> Report
  -> Ok(report)
  ```
  `BuildSpecLoader` is a **pure, total factory call — never an error step** (no `Result`, no
  short-circuit): it must run *after* `NewConfig` so `cfg.Settings.Timeout` is the real,
  already-validated value (never a hardcoded constant — this is why the step sits *inside* the
  pipe, not at wiring time; EMULATION.md S15 guards this exact ordering).
  - antecedent: a well-formed `Invocation`.
  - consequent: success → `Report`; failure → the first short-circuiting step's `Error`,
    untransformed. `CompareContracts` never returns an error on the domain axis —
    `incompatible` is a legitimate `Outcome` value, not an `Err`.
- unit tests: **none** (pipe of already-tested parts — exit codes asserted by component
  scenarios; the factory step is total, so it adds no branch to test).
- component scenario(s) to green: all 8 scenarios end-to-end (with wiring, ticket 23) — greened
  later by @fagan. Scenario 8 (`TIMEOUT_ERROR`) is reachable precisely because of the late
  `BuildSpecLoader` step.

**Dependencies:** `ConfigStore.Load`(06), `NewConfig`(07), `ContractStore.Load`(08),
`NewConsumedContract`(09), the `SpecLoader` interface(11), `BuildSpecLoader`(13),
`DeriveProviderChannels`(14), `NewComparison`(15), `CompareContracts`(18), `FoldReport`(19), the
`ReportWriter` interface(20), `BuildReportWriter`(21); shared types + `Clock`(03).

**Subagent instruction:** declare `Deps` + implement `ProcessValidate` in
`internal/validate/head.go` (new file) → `go build`/`vet` → done. Touch no other module; do NOT
run the component harness or drive scenarios GREEN.

**Verify:** `go build ./internal/validate/... && go vet ./internal/validate/...`.

**Acceptance:** package `validate` builds and vets clean; `Deps` + `ProcessValidate` present,
wired exactly per the pipe above (late `BuildSpecLoader` call after `NewConfig`); no units.
Component GREEN is the @fagan acceptance step.
