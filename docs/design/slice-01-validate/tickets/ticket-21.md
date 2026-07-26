---
id: 21
type: module
slice: slice-01-validate
blocked_by: [01, 02, 03, 20]
inputs: [docs/design/slice-01-validate/contracts.md, docs/design/slice-01-validate/module-tree.md]
outputs: [internal/validate/logic.go, internal/validate/buildreportwriter_test.go]
io: none
skills: []
---

### TICKET 21 — slice-01-validate/BuildReportWriter: whether/where-to-persist factory

**io:** none → skills: [] (factory — **unit-tested by formula**).

**Context (only this module — APPEND to `internal/validate/logic.go`):**
- contract (`contracts.md` §BuildReportWriter): `BuildReportWriter(settings: Settings) ->
  ReportWriter`. Constructs the writer, encapsulating **whether/where** to persist (Store-style:
  the head never branches on `save_json_report` itself) — returns a no-op `ReportWriter` when
  `settings.SaveJSONReport == false`, else a writer targeting `settings.JSONReportFile`. Deps = —.
  - antecedent: —.
  - consequent: a `ReportWriter` (ticket 20's interface) — no-op internally when
    `save_json_report == false`.
- **unit tests: 2** (`contracts.md` §4 formula) — 1 happy + 1 branch: `save_json_report == false`
  → no-op writer (distinguishable behavior at `.Write`, but the *construction* branch itself is
  the unit).
- component scenario(s) to green: none directly (underlies scenario 1's "report file written"
  assertion, greened later).

**Dependencies:** `Settings` type (ticket 03); `ReportWriter` interface + concrete writer (ticket
20).

**Subagent instruction:** write the 2 unit tests in
`internal/validate/buildreportwriter_test.go` → implement `BuildReportWriter` in
`internal/validate/logic.go` (append) → run units → green → done. Touch no other module.

**Verify:** `go build ./internal/validate/... && go vet ./internal/validate/... && go test ./internal/validate/... -run TestBuildReportWriter`.

**Acceptance:** package `validate` builds/vets clean; 2 unit tests green.
