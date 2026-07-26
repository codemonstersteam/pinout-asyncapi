---
id: 05
type: module
slice: slice-01-validate
blocked_by: [01, 02, 03, 04]
inputs: [docs/design/slice-01-validate/contracts.md, docs/design/slice-01-validate/module-tree.md, api-specification/report.schema.json]
outputs: [internal/validate/adapter.go]
io: none
skills: []
---

### TICKET 05 — slice-01-validate/cli.ResolveExitCode: Result → exit code (egress adapter)

**io:** none → skills: [] (mechanical Result→exit mapping, `cli-io` grid). **Not unit-tested** —
asserted by component scenarios.

**Context (only this module — APPEND to the existing `internal/validate/adapter.go` from ticket
04; do not touch `Parse`):**
- contract (`contracts.md` §cli.ResolveExitCode): `ResolveExitCode(res: Result[Report, Error]) ->
  int`. Grid (`module-tree.md` §5): `Ok(compatible)`→0; `Ok(incompatible)`→1; `ErrConfigInvalid`→2;
  `ErrFileNotFound|ErrParseError|ErrHTTPError|ErrTimeoutError`→3. Deps = —.
  - antecedent: none (total function over the closed `Error` taxonomy).
  - consequent: exactly one of `{0,1,2,3}`.
- **Note:** this function only maps `Result`→`int`. **Printing** the report to stdout / logs to
  stderr + `os.Exit` is done by `main` (wiring, ticket 23), not here.
- unit tests: **none** (mechanical adapter — exit codes asserted by component scenarios).
- component scenario(s) to green: contributes the exit code to all 8 scenarios — greened later.

**Dependencies:** sentinels + `Report` type (ticket 03).

**Subagent instruction:** implement `cli.ResolveExitCode` in `internal/validate/adapter.go`
(append) → `go build`/`vet` → done. Touch no other module.

**Verify:** `go build ./internal/validate/... && go vet ./internal/validate/...`.

**Acceptance:** package `validate` builds and vets clean with `ResolveExitCode` present; no units.
Component GREEN is @fagan's step.
