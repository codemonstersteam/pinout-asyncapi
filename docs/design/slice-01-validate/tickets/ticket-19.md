---
id: 19
type: module
slice: slice-01-validate
blocked_by: [01, 02, 03, 18]
inputs: [docs/design/slice-01-validate/contracts.md, docs/design/slice-01-validate/module-tree.md, api-specification/report.schema.json]
outputs: [internal/validate/logic.go, internal/validate/foldreport_test.go]
io: none
skills: []
---

### TICKET 19 — slice-01-validate/FoldReport: assemble the canon-`1.1` Report

**io:** none → skills: [] (pure DTO assembly — **unit-tested by formula**).

**Context (only this module — APPEND to `internal/validate/logic.go`):**
- contract (`contracts.md` §FoldReport): `FoldReport(outcome: Outcome, clock: Clock) -> Report`.
  Assembles the canon-`1.1` `Report` (`report.schema.json`): `compatible ⇔ errors == []`,
  `errors[]` (from `Violations`), `provenance` (echoed), `uncovered_channels[]`, `generated_at`
  (from `clock.Now()`, called **exactly once** — the whole slice's only clock read, D10 — the core
  never reads the system clock directly), `validator: "pinout-asyncapi"` / `interaction: "async"`
  build-time constants. Deps = `Clock` (ticket 03 — the port; calls `.Now()` exactly once here).
  - antecedent: a valid `Outcome` (ticket 18).
  - consequent: `Report`, total — no failure branch.
- **unit tests: 2** (`contracts.md` §4 formula) — 1 happy + 1 branch: `compatible ⇔ errors == []`
  both directions (empty violations → `true`; non-empty → `false`). Use a fake `Clock` returning a
  fixed `time.Time` so the test is deterministic.
- component scenario(s) to green: none directly (underlies scenario 1's `generated_at` field and
  every scenario's report shape, greened later by @fagan).

**Dependencies:** `Outcome`/`Report`/`Clock` types (ticket 03); `CompareContracts` (ticket 18, for
realistic `Outcome` fixtures).

**Subagent instruction:** write the 2 unit tests in
`internal/validate/foldreport_test.go` → implement `FoldReport` in `internal/validate/logic.go`
(append) → run units → green → done. Touch no other module.

**Verify:** `go build ./internal/validate/... && go vet ./internal/validate/... && go test ./internal/validate/... -run TestFoldReport`.

**Acceptance:** package `validate` builds/vets clean; 2 unit tests green.
