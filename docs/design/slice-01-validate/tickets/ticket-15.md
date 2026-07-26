---
id: 15
type: module
slice: slice-01-validate
blocked_by: [01, 02, 03, 07, 09, 14]
inputs: [docs/design/slice-01-validate/contracts.md, docs/design/slice-01-validate/module-tree.md]
outputs: [internal/validate/logic.go, internal/validate/newcomparison_test.go]
io: none
skills: []
---

### TICKET 15 — slice-01-validate/NewComparison: unite Config+Contract+Channels (uniting constructor)

**io:** none → skills: [] (uniting constructor — the **sanctioned** plural-input exception,
`module-tree.md` §3 note 1 — **unit-tested by formula**).

**Context (only this module — APPEND to `internal/validate/logic.go`):**
- contract (`contracts.md` §NewComparison): `NewComparison(cfg: Config, contract:
  ConsumedContract, pchans: ProviderChannels) -> Result[Comparison, Error]`. Unites the three
  already-valid parts into one valid `Comparison`; performs the **one** cross-artifact check that
  can only run here (FRD §4.1: every address in `cfg.Consumer.Channels` must appear in
  `contract.Channels`). Deps = —.
  - antecedent: every address in `cfg.Consumer.Channels` is present among
    `contract.Channels[].Address`.
  - consequent: Ok `Comparison`; failure → `ErrConfigInvalid` (`CONFIG_ERROR`) — still before any
    report is written.
- **unit tests: 2** (`contracts.md` §4 formula) — 1 happy + 1 branch: a
  `cfg.Consumer.Channels` address absent from `contract.Channels`.
- component scenario(s) to green: none directly (this `CONFIG_ERROR` path is UNIT-covered here;
  it shares the code with `ConfigStore.Load`/`NewConfig`'s failures, `contracts.md` §2's "three
  places, one code" note — component scenario 2 is a representative fixture, not this exact
  branch).

**Dependencies:** `Config` (ticket 07), `ConsumedContract` (ticket 09), `ProviderChannels`
(ticket 14) types; `Comparison` type + `ErrConfigInvalid` sentinel (ticket 03).

**Subagent instruction:** write the 2 unit tests in
`internal/validate/newcomparison_test.go` → implement `NewComparison` in
`internal/validate/logic.go` (append) → run units → green → done. Touch no other module.

**Verify:** `go build ./internal/validate/... && go vet ./internal/validate/... && go test ./internal/validate/... -run TestNewComparison`.

**Acceptance:** package `validate` builds/vets clean; 2 unit tests green.
