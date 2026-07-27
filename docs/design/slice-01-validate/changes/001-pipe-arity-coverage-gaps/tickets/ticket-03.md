---
id: 03
type: module
slice: slice-01-validate
blocked_by: [01, 02]
inputs: [docs/design/slice-01-validate/changes/001-pipe-arity-coverage-gaps/change-delta.md, docs/design/slice-01-validate/changes/001-pipe-arity-coverage-gaps/contracts.md, cmd/app/main_test.go, cmd/app/main.go, internal/validate/adapter.go, internal/validate/io_report.go]
outputs: [cmd/app/main_test.go]
io: none
skills: []
---

### TICKET 03 — slice-01-validate/cli.ResolveExitCode: pin the two branches nothing asserts

**io:** none → skills: [] (binary-level test additions; **no production code is changed**)

**Runs before the reshape (04/05/06), on unmodified production code.** These cases pin behaviour the
refactor must preserve, so they belong in the regression envelope, not after it. `internal/validate/`
and `cmd/app/main.go` are **not** edited by this ticket.

**Surface under test:** `cli.ResolveExitCode` (`internal/validate/adapter.go:32`) — documented in
`contracts.md` as **total over ANY `error`**, its `default:` arm being the deliberate
out-of-taxonomy fallback → **3** (D3). No adapter unit-test file is created: **C0 stands verbatim —
`internal/validate/adapter_test.go` MUST NOT exist**; the adapter is proven **through the binary
only**, and the binary path *is* production, so every error is wrapped exactly as production raises
it (`contracts.md` §5 "C2").

**Scope, verified against the current `cmd/app/main_test.go` (do not assume more):** the file today
holds exactly `TestVersionCmd` and `TestValidateCmd_ConfigFileNotFound`. The 7-row
`ResolveExitCode` grid (`contracts.md` §5, C2) is therefore already **6/7** covered without any new
Go test — row 1 ← component scenario 1; row 3 ← `TestValidateCmd_ConfigFileNotFound` + scenario 2;
rows 4–7 ← scenarios 3a/3b/4a/4b/4c/4d; **row 2 (exit 1) is closed by ticket-01's scenario 9**, not
here. So this ticket adds **exactly 3 cases and no grid rows**:

#### C4 — the out-of-taxonomy fallback (1 case)

`root.Execute()` with `validate <cfg>` where `<cfg>` is a config assembled at run time in
`t.TempDir()` over the repo's `component-tests/fixtures/validate/good/{consumed-contract,provider-asyncapi}.yaml`
(reachable, compatible pair — resolve paths relative to the test file), with `save_json_report: true`
and **`json_report_file` inside a non-existent directory**. `os.WriteFile` then fails at
`internal/validate/io_report.go:53`; that error wraps **no** sentinel from `errors.go`.

- Assert: the returned error is an `*exitError` with `code == 3`.
- Under the fault (`io_report.go` "helpfully" wrapping the write failure as `%w: ErrConfigInvalid`,
  or `ResolveExitCode` gaining an arm that narrows its totality) the case observes **2** → RED.
- **MUST assert nothing else.** Today's stdout body on this path carries `code: ""` and no `subject`
  — schema-**invalid** against `report.schema.json`. Pinning or fixing that exceeds the `patch`
  envelope; it is recorded as new debt (`change-delta.md` §3 row B), **not** repaired here.

#### C1 — argc ≠ 1 never reaches the pipe (2 cases)

`root.Execute()` with `[]string{"validate"}` (0 args) and with `[]string{"validate", "a", "b"}`
(2 args).

- Assert in both: `err != nil` **and** it is **NOT** an `*exitError` (i.e. `errors.As(err,
  &ee) == false`) — the invocation is rejected by cobra's `Args` validator before `cli.Parse` and
  the pipe ever run.
- Defense-in-depth is intended: the invariant is held by **two** guards (`cobra.ExactArgs(1)` in
  `newValidateCmd` and `Parse`'s own `len(args) != 1` guard). Dropping *one* keeps the invariant and
  these cases correctly stay green — that is the invariant holding, not insensitivity. **Neither
  guard is removed, and `main.go` is not edited** (`change-delta.md` §2, "Explicitly NOT touched").

**Dependencies:** ticket-01 (component ticket — RED-first edge), ticket-02 (both anchors captured
before any later signature edit).

**Subagent instruction:** append the 3 cases to `cmd/app/main_test.go` → run them on unmodified code
→ green → done. Do not create `internal/validate/adapter_test.go`. Do not edit `main.go`,
`adapter.go` or `io_report.go`.

**Verify:** `go build ./... && go vet ./... && go test ./cmd/...`

**Acceptance:**

- [ ] `cmd/app/main_test.go` holds 3 new cases (C4 ×1, C1 ×2) alongside the 2 pre-existing ones; no
      existing case edited or weakened.
- [ ] `internal/validate/adapter_test.go` does **not** exist (C0).
- [ ] `git diff --stat cmd/app/main.go internal/validate/` = 0 files changed by this ticket.
- [ ] `go test ./cmd/...` GREEN on unmodified production code.
