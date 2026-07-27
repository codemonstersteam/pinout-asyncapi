---
id: 02
type: module
slice: slice-01-validate
blocked_by: [01]
inputs: [docs/design/slice-01-validate/changes/001-pipe-arity-coverage-gaps/change-delta.md, docs/design/slice-01-validate/changes/001-pipe-arity-coverage-gaps/contracts.md, internal/validate/head.go, internal/validate/report_json_shape_test.go, component-tests/fixtures/validate/good/config.yaml]
outputs: [internal/validate/processvalidate_determinism_test.go, internal/validate/testdata/report.golden.json]
io: none
skills: []
---

### TICKET 02 — slice-01-validate/ProcessValidate: the D10 determinism anchor (golden, pre-refactor)

**io:** none → skills: [] (test artifact + golden data; **no production code is touched**)

**THIS TICKET RUNS ON UNMODIFIED PRODUCTION CODE, BEFORE ANY SIGNATURE IS EDITED.** It is the
second of the two byte anchors (`change-delta.md` §3 row E, `module-tree.md` §4 closing note); the
first is ticket-01's stdout baseline. It **blocks every reshape ticket (04, 05, 06)** for exactly
this reason: regenerating the golden from post-refactor output would void the proof and turn the
whole change into an unverified restructure. Do **not** edit `logic.go`, `head.go` or `domain.go`
here.

**Surface being anchored (unchanged by this change, BRD N4):**
Signature: `ProcessValidate(inv: Invocation, deps: Deps) -> Result<Report, Error>` — the head must be
callable **identically** before and after the reshape, which is what makes this test a valid
before/after comparator.

**What this ticket writes — one Go test file + one golden artifact:**

`internal/validate/processvalidate_determinism_test.go`, driving `ProcessValidate` over the `good/`
pair with a **frozen** clock:

1. Assemble the config **at run time** in `t.TempDir()` over the repo's existing files
   `component-tests/fixtures/validate/good/consumed-contract.yaml` and
   `.../good/provider-asyncapi.yaml` (resolve the paths relative to the test file). **MUST NOT fork
   those bytes** — reading the same files is what keeps this anchor and ticket-01's I2 baseline from
   drifting apart. `save_json_report: false`; `spec_path` (file branch, no HTTP).
2. `Deps{Clock: <counting frozen clock>, HTTPClient: <the same client register.go builds, or a
   default one — the file branch never uses it>}`.
3. **Assertion 1 (I1a — one clock read):** the fake `Clock` counts `Now()` calls and the test asserts
   **exactly 1** call across the whole `ProcessValidate` run. A `Reporter` reading the clock twice,
   or per-violation, fails here. (`foldreport_test.go`'s existing `fixedClock` cannot see this — it
   returns the same instant every time and counts nothing.)
4. **Assertion 2 (I1c — full-byte golden):** marshal the returned `Report` with the **same writer the
   binary uses for stdout** (`internal/shared/report`.`WriteJSON`) and compare the **full bytes** —
   **including** `generated_at` (it is deterministic here: the clock is frozen) — against
   `internal/validate/testdata/report.golden.json`. No normalization, no struct comparison.
5. **Doc comment (MUST, verbatim intent from `contracts.md` §ProcessValidate):** mark the file a
   **"D10 determinism regression anchor, not a head unit test"** — the same documented non-formula
   status `internal/validate/report_json_shape_test.go` already carries. It adds **no** row to
   `contracts.md` §4; the head is still not unit-tested (C0/Q2 stand verbatim).

**Capturing `testdata/report.golden.json`:** write the test first with the golden absent, run it once
with a capture switch (e.g. `-run TestProcessValidateDeterminism -update` guarded by a `testing`
flag, or a one-off `os.WriteFile` you then remove) against the **current, unmodified** code, inspect
the result (canon-`1.1`, `compatible: true`, `errors: []`, frozen `generated_at`), commit it, then
confirm the test passes reading it. **The golden must never be regenerated after ticket-04/05/06.**

**Dependencies:** ticket-01 (the component suite already carries its own baseline; both anchors are
taken pre-refactor). No production module.

**Subagent instruction:** write the test → capture the golden from the **current** code → verify the
test is GREEN on unmodified code → done. Touch no production file.

**Verify:** `go build ./... && go vet ./... && go test ./internal/validate/... ` — the whole
`internal/validate` package suite (49 existing unit tests + this anchor) GREEN, on **unmodified**
production code.

**Acceptance:**

- [ ] `internal/validate/testdata/report.golden.json` exists and was captured from the pre-refactor
      binary; `git diff --stat internal/validate/*.go` shows **0** production files changed by this
      ticket.
- [ ] The test asserts exactly **1** `Now()` call per run and full-byte equality against the golden.
- [ ] The file's doc comment marks it a D10 determinism regression anchor, not a head unit test.
- [ ] `go test ./internal/validate/...` GREEN before any signature edit.
