---
id: 23
type: module
slice: slice-01-validate
blocked_by: [01, 02, 03, 04, 05, 06, 07, 08, 09, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22]
inputs: [docs/design/slice-01-validate/module-tree.md, docs/design/slice-01-validate/contracts.md]
outputs: [internal/validate/register.go, cmd/app/main.go]
io: none
skills: []
---

### TICKET 23 — slice-01-validate/wiring: concrete Deps + mount the command (exposes the CLI)

**io:** none → skills: [] (wiring only — implements NO module logic; single concern = expose the
command by constructing the concrete `Deps`, ticket 22).

**Context (only wiring — two files):**
- `internal/validate/register.go` — construct the concrete `Deps` for `ProcessValidate` (ticket
  22): real `ConfigStore` (ticket 06), `ContractStore` (ticket 08), a `systemClock` implementing
  `Clock` (`type systemClock struct{}; func (systemClock) Now() time.Time { return time.Now() }`
  — the **only** production clock read, D10), a real `*http.Client` (sane default timeout at the
  transport level; the *domain* timeout comes from `cfg.Settings.Timeout` inside the pipe, not
  here — do not hardcode a provider timeout in this file). No domain logic.
- `cmd/app/main.go` — the live wiring per `module-tree.md` §4: `cli.Parse(os.Args)` → build
  `Deps` (register.go) → `ProcessValidate(inv, deps)` → `code := cli.ResolveExitCode(res)`;
  **always print the report JSON to stdout** (machine channel), diagnostics/logs to **stderr**;
  `os.Exit(code)`. Keep the binary at `cmd/app/` (no `cmd/<slug>/` rename). Command surface:
  `validate <config.yaml>` + `version`/`--help`.
- unit tests: **none** (wiring).
- component scenario(s) to green: this ticket makes all 8 scenarios reachable end-to-end.
  **Greening + `@wip`-removal is the @fagan acceptance step, NOT this ticket's deliverable.**

**Dependencies:** every module ticket (03–22) — `cli.Parse`(04), `cli.ResolveExitCode`(05), every
I/O object (06, 08, 11, 12, 20), `BuildSpecLoader`(13)/`BuildReportWriter`(21) factories,
`ProcessValidate` + its `Deps`(22).

**Subagent instruction:** implement `register.go` (concrete `Deps` + `systemClock`) → wire
`cmd/app/main.go` (`cli.Parse` → `Deps` → `ProcessValidate` → `cli.ResolveExitCode` →
print/exit) → `go build ./...` → done. Do NOT implement module logic; do NOT run the component
harness to GREEN (that is @fagan).

**Verify:** `go build ./... && go vet ./...` (whole module builds).

**Acceptance:** `go build ./...` green; `pinout-asyncapi validate <config.yaml>` runs the real
pipe, report printed to stdout, exit code from `cli.ResolveExitCode`. Slice closure (remove
`@wip` + component GREEN + verify every `TASK.md` DoD item) is the **@fagan acceptance step**.
