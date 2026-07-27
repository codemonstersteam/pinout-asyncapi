---
id: 06
type: module
slice: slice-01-validate
blocked_by: [02, 05]
inputs: [docs/design/slice-01-validate/changes/001-pipe-arity-coverage-gaps/contracts.md, docs/design/slice-01-validate/changes/001-pipe-arity-coverage-gaps/module-tree.md, docs/design/slice-01-validate/changes/001-pipe-arity-coverage-gaps/adr/001-factory-and-product-method-are-two-nodes.md, internal/validate/logic.go, internal/validate/head.go, internal/validate/foldreport_test.go]
outputs: [internal/validate/logic.go, internal/validate/head.go, internal/validate/foldreport_test.go, internal/validate/buildreporter_test.go]
io: none
skills: []
---

### TICKET 06 — slice-01-validate/BuildReporter + Reporter.Fold: enclose the clock port

**io:** none → skills: [] (pure logic; edit-in-place reshape of an existing, shipped node)

**Nature: reshape, behaviour-preserving (`patch`).** `FoldReport(outcome, clock)`
(`internal/validate/logic.go:1074`) takes a data entity **and** a port. The port moves to
construction; `Fold` keeps the one data input. Tree nodes **18 + 19** (`module-tree.md` §3); one
reshape, one ticket — the factory and its product method cannot compile apart (ADR-001 splits them as
*design* nodes, not as tickets).

**SEQUENCING — MUST hold:** `blocked_by` names **ticket-02** directly and not only through the chain.
This is the node the two byte anchors exist to protect (`change-delta.md` §3 rows D/E): the golden and
the stdout baseline were captured from `9eef9e9`, **before** this signature was touched.
**Regenerating either artifact in this ticket voids the proof of the whole change** — if the golden
does not match after the edit, that is a real defect, not a stale artifact. STOP and report instead.

**Contract (`contracts.md` §BuildReporter + §Reporter.Fold):**

- Factory — Signature: `BuildReporter(c: Clock) -> Reporter`
  - Input: the `Clock` port (the head passes `deps.Clock`). Deps: —. `io: none`.
  - Antecedent: a non-nil `Clock`.
  - Consequent: a `Reporter` with the clock in an **unexported** field. **No failure branch** — this
    totality is why hoisting the bind ahead of the chain is behaviour-preserving. Performs no I/O; it
    is a collaborator, not an I/O object.
- Product method — `(r Reporter) Fold(outcome Outcome) Report`
  - Input: `Outcome` — **one** data entity. Deps: — (the clock is captured in the receiver).
  - What it does, unchanged: assembles the canon-`1.1` report — `compatible ⇔ errors == []`,
    `errors[]` from `Violations`, echoed `provenance`, `uncovered_channels[]`, `generated_at` from
    the bound clock, `validator`/`interaction` build-time constants.
  - Consequent: `Report`, **total** (no failure branch), **byte-identical** to the pre-change
    `FoldReport` for the same `Outcome` and the same instant.

**D10, tightened not weakened (BRD I1a/I1b):** the slice's **only** clock read now lives *inside*
`Reporter`, reachable only through the port bound at `BuildReporter`. After this ticket
`grep -rn 'time.Now()' internal/validate/` must still match **only** `register.go`, and ticket-02's
counting fake must still observe **exactly 1** `Now()` call per run.

Both live in `internal/validate/logic.go`, the `Reporter` struct beside its factory (`module-tree.md`
§6 — **no file is added, removed or renamed**).

**Call-site rewiring (same ticket — the tree must compile and stay green):** in
`internal/validate/head.go`:

```go
reporter := BuildReporter(deps.Clock)   // pre-chain bind block, after NewConfig (D10)
…
report := reporter.Fold(outcome)        // chain: one data entity
```

Ticket-07 normalizes the final order of that bind block; do not move
`BuildSpecLoader`/`BuildReportWriter` here.

**Unit tests (`contracts.md` §4):**

- `internal/validate/foldreport_test.go` — **mechanical call-site updates only**:
  `FoldReport(outcome, clk)` → `BuildReporter(clk).Fold(outcome)`. **0 assertions deleted or
  weakened** (BRD N5); both cases (happy + `compatible ⇔ errors == []` in both directions) stay
  exactly as they are. Its existing `fixedClock` stays as-is — the call-counting fake is ticket-02's
  anchor, not this file's job.
- `internal/validate/buildreporter_test.go` — **new**, **N = 1** (happy only; the factory binds a port
  and has no antecedent branch). Follow the shape of the shipped `buildreportwriter_test.go`.

**Dependencies:** ticket-02 (the anchors — see SEQUENCING above), ticket-05 (serializes the
`logic.go`/`head.go` edits; carries the ticket-01 component edge transitively).

**Subagent instruction:** split the node in `logic.go` → rewire the one call site in `head.go` →
update `foldreport_test.go` mechanically → add `buildreporter_test.go` → run the package suite + the
anchors → green → done. **Do not regenerate `internal/validate/testdata/report.golden.json`.** Touch
no other module.

**Verify:** `go build ./... && go vet ./... && go test ./internal/validate/... && go test ./cmd/...`

**Acceptance:**

- [ ] `FoldReport` no longer exists; `BuildReporter` + `Reporter.Fold` do, with the consequent above
      unchanged.
- [ ] `reporter.Fold(outcome)` in `head.go` takes exactly **one** data argument; the clock is passed
      nowhere but `BuildReporter`.
- [ ] `grep -rn 'time.Now()' internal/validate/` matches **only** `register.go` (I1b).
- [ ] `foldreport_test.go`: same cases, same assertions, call sites updated only.
      `buildreporter_test.go` exists with 1 test.
- [ ] **The anchor still passes against the UNCHANGED golden** — ticket-02's test GREEN with
      `git diff --stat internal/validate/testdata/` = 0 files changed (I1c).
- [ ] **Regression envelope holds (E1):** `go test ./internal/validate/... ./cmd/...` GREEN.
