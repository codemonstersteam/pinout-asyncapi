---
id: 20
type: module
slice: slice-01-validate
blocked_by: [01, 02, 03]
inputs: [docs/design/slice-01-validate/contracts.md, docs/design/slice-01-validate/module-tree.md]
outputs: [internal/validate/io_report.go]
io: none
skills: []
---

### TICKET 20 — slice-01-validate/ReportWriter.Write: write JSON report to disk (I/O pipe)

**io:** none → skills: [] (filesystem write pipe). Pure pipe, **not unit-tested**.

**Context (only this module):**
- **Declare the `ReportWriter` interface HERE**: `type ReportWriter interface { Write(report
  Report) (Report, error) }`.
- contract (`contracts.md` §ReportWriter.Write): `(w ReportWriter) Write(report: Report) ->
  Result[Report, Error]`. Writes the report bytes to the configured path (or no-op, per how it was
  constructed by `BuildReportWriter`, ticket 21); **always returns the report unchanged** for the
  pipe to continue (ROP pass-through — never partially mutates). Deps = — (OS filesystem
  encapsulated).
  - antecedent: —.
  - consequent: Ok the same `Report` unchanged. **A disk-write failure here has no code in the
    frozen error taxonomy** (`report.schema.json` `x-exit-codes` has none for it) — out of scope
    for this design; not a new component scenario (frozen contracts are not modified by this
    stage, `contracts.md` §3 note).
- unit tests: **none** (I/O pipe).
- component scenario(s) to green: the happy scenario (1) asserts the file **is** written when
  `save_json_report` — greened later.

**Dependencies:** `Report` type (ticket 03).

**Subagent instruction:** declare `ReportWriter` interface + implement the concrete writer +
`Write` in `internal/validate/io_report.go` (new file) → `go build`/`vet` → done. Touch no other
module.

**Verify:** `go build ./internal/validate/... && go vet ./internal/validate/...`.

**Acceptance:** package `validate` builds and vets clean; `ReportWriter` interface + concrete
writer present; no units (I/O pipe).
