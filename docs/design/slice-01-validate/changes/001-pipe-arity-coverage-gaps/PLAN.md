# PLAN — change 001-pipe-arity-coverage-gaps (slice-01-validate, lane `patch`)

> Plan-index for Gate #1 (operator acceptance). Assembled by `wirth-planner` from the finished
> change-scoped design package below. This file designs nothing — it indexes paths and summarizes
> for the operator's approve/reject decision. SemVer lane: `patch` — this plan lives in the
> **change's own folder**, not the slice's greenfield `PLAN.md` (`docs/design/slice-01-validate/PLAN.md`,
> untouched by this change).

## 1. Design package (path index)

| Artifact | Path |
|---|---|
| Change delta (what/why/ripple, fitness check) | [`change-delta.md`](./change-delta.md) |
| Module tree + head-pipe pseudocode (change-scoped, 21 nodes) | [`module-tree.md`](./module-tree.md) |
| Module contracts + unit-test formula (51) + component-scenario set (N=9) | [`contracts.md`](./contracts.md) |
| C4 (C2 Container + C3 Component, change-scoped) | [`c4.md`](./c4.md) |
| ADR-001 — factory/product-method = two tree nodes | [`adr/001-factory-and-product-method-are-two-nodes.md`](./adr/001-factory-and-product-method-are-two-nodes.md) |
| Tickets (7, dependency-ordered) | [`tickets/`](./tickets/) |
| Affected slice — canonical (frozen until run-close) design package | [`../../module-tree.md`](../../module-tree.md) · [`../../contracts.md`](../../contracts.md) · [`../../c4.md`](../../c4.md) · [`../../use-case.md`](../../use-case.md) |
| Affected slice's own plan (untouched by this change) | [`../../PLAN.md`](../../PLAN.md) |
| Frozen machine contracts (untouched, N1) | [`../../../../../api-specification/config.schema.json`](../../../../../api-specification/config.schema.json) · [`consumed-contract.schema.json`](../../../../../api-specification/consumed-contract.schema.json) · [`report.schema.json`](../../../../../api-specification/report.schema.json) |
| BRD (source of this change) | [`../../../../../.agent/planner/brd.md`](../../../../../.agent/planner/brd.md) |
| Source business requirement | [`../../../../../debt/task-001.md`](../../../../../debt/task-001.md) |
| Route classification | [`../../../../../.agent/planner/mode`](../../../../../.agent/planner/mode), [`change-dir`](../../../../../.agent/planner/change-dir) |

Package scope: one existing slice (`slice-01-validate`, package `internal/validate/`), no new
external input, no new module (behaviour axis), lane `patch` per `wirth-triage` (`.agent/planner/mode`).
Baseline commit: **`9eef9e9`** — both byte anchors (I2 stdout baseline, I1c `ProcessValidate` golden)
MUST be captured from this commit before any signature is touched (change-delta.md §5, ordering
constraint).

## 2. Gate #1 summary

### 2.1 What changes + why

Three nodes of the `ProcessValidate` pipe take **two** data arguments instead of one
(`FoldReport(outcome, clock)`, `NewConsumedContract(raw, expectedConsumer)`,
`NewComparison(cfg, contract, pchans)`), and two behavioural branches of the CLI are asserted by
**no test anywhere**: the primary verdict exit `1` (incompatible) and the out-of-taxonomy fallback
exit `3`. This change:

1. Rebinds the three call sites to the already-established bind-then-chain idiom —
   `BuildReporter(clock).Fold(Outcome)`, `BuildContractParser(consumerName).Parse(RawContract)`,
   `NewComparison(ComparisonInput{Config, Contract, ProviderChannels})` (the one genuine
   three-stream join, materialized as a named type rather than removed).
2. Closes both coverage holes: a new component scenario 9 (exit 1) and 3 new `cmd/app/main_test.go`
   cases (2× argc-breach short-circuit, 1× out-of-taxonomy fallback).

**Why now, and why `patch` holds.** The arity edit is a pure restructure — by construction it changes
no observable output, so alone it is *unfalsifiable*. What makes it safe is a pair of byte-level
anchors captured from the **pre-refactor** binary (I2 stdout baseline on the `good/` fixture; I1c
`ProcessValidate` golden under a frozen `Clock`) — both MUST be captured from `9eef9e9` before the
first signature is edited (irreversible sequencing constraint, `change-delta.md` §1/§5, tickets
01/02 `blocked_by: []`/`[01]`). What makes it *worth* doing now is that the same pass buys back the
two grid rows the suite could never see. Public surface (`api-specification/*.schema.json`, argv,
exit-code grid, report bytes) is untouched — `N1`/`N2` (change-delta.md §1, §6).

### 2.2 Design decision this change settles (ADR-001)

`change-delta.md` §5 left one question open for the design stage: does a factory + its product
method occupy **one** tree row or **two**? Settled **two**, per the tree's own shipped convention
(`BuildSpecLoader`/`FileSpecLoader.Load`, `BuildReportWriter`/`ReportWriter.Write` — both already
split). Consequence: the module tree goes **19 → 21** nodes and `contracts.md` §4's unit-test total
goes **49 → 51** (two branchless factory rows, `BuildReporter`/`BuildContractParser`, N=1 each) —
this **corrects the BRD's stale pinned total of 49** (BRD N7/D1 assumed a rename-in-place; ADR-001
supersedes it). No adapter row, no head row, no I/O row is added — the frozen "head/I-O/adapters are
not unit-tested" rule (C0/Q2) stands verbatim. Rationale: [`adr/001-factory-and-product-method-are-two-nodes.md`](./adr/001-factory-and-product-method-are-two-nodes.md).

### 2.3 Head-pipe functional block (bind-then-chain, `ProcessValidate` — verbatim from `module-tree.md` §4)

```
ProcessValidate(inv: Invocation, deps: Deps) -> Result<Report, Error>:

  # --- stage 0: the config, which every bind below depends on ---
  | ConfigStore.Load(inv.ConfigPath)                       -> RawConfig         # fs        [CONFIG_ERROR -> 2]
  | NewConfig(raw)                                          -> Config           # validate  [CONFIG_ERROR -> 2]

  # --- bind block: environment + validated config scalars -> collaborators ---
  # not pipe steps: no data flows through them, no I/O, no failure branch (all four are total)
  parser   := BuildContractParser(cfg.Consumer.Name)                            # config scalar -> collaborator
  loader   := BuildSpecLoader(cfg.Provider, cfg.Settings.Timeout)               # strategy pick (D … EMULATION S15)
  reporter := BuildReporter(deps.Clock)                                         # clock port -> collaborator (D10)
  writer   := BuildReportWriter(cfg.Settings)                                   # whether/where to persist

  # --- chain: every step takes exactly one data entity ---
  | ContractStore.Load(cfg.Consumer.ConsumedContractPath)   -> RawContract      # fs        [FILE_NOT_FOUND -> 3]
  | parser.Parse(raw)                                       -> ConsumedContract # validate  [PARSE_ERROR -> 3]
  | loader.Load()                                           -> ProviderSpec     # fs XOR http
  |     (internally: InlineServerRefs -> parser.FromYAML(...).Process())       [FILE_NOT_FOUND | PARSE_ERROR | HTTP_ERROR | TIMEOUT_ERROR -> 3]
  | DeriveProviderChannels(spec)                            -> ProviderChannels # projection: 2 defaults expanded (D3)
  | NewComparison(ComparisonInput{cfg, contract, pchans})   -> Comparison       # unite      [CONFIG_ERROR -> 2] (channel-scope cross-check)
  | CompareContracts(comparison)                            -> Outcome          # CORE: R1..R9 fold, never short-circuits across channels
  | reporter.Fold(outcome)                                  -> Report           # canon 1.1; the slice's only clock read (D10)
  | writer.Write(report)                                    -> Report           # iff save_json_report; pass-through
  -> Ok(report)

then in main: code := cli.ResolveExitCode(result); report ALWAYS -> stdout; logs -> stderr; os.Exit(code)
```

`ProcessValidate`'s own signature `(Invocation, Deps) -> Result<Report, Error>` and
`Deps{Clock, HTTPClient}` are **unchanged** (BRD N4) — the I1c golden depends on the head being
callable identically before and after. Full pseudocode + rationale: [`module-tree.md`](./module-tree.md) §4.

### 2.4 Affected modules (behaviour axis: 0 new, 3 reshaped, 3 documentation-only)

| # (new tree) | Module | File | Nature |
|---|---|---|---|
| 7–8 | `NewConsumedContract` → `BuildContractParser` + `ContractParser.Parse` | `logic.go:120` | Reshape — `expectedConsumer` moves to construction |
| 13 | `NewComparison` | `logic.go:576` | Join named — `ComparisonInput{Config, Contract, ProviderChannels}` |
| 17–19 (was 17) | `FoldReport` → `BuildReporter` + `Reporter.Fold` | `logic.go:1074` | Reshape — `Clock` port moves to construction |
| 2 | `ProcessValidate` (head) | `head.go:35` | Rewiring only — bind-then-chain; signature unchanged |
| 3 | `cli.ResolveExitCode` | `adapter.go:32` | No code change — documentation (totality, D3) + newly discriminated by tests |
| 21 (was 19) | `ReportWriter.Write` | `io_report.go:42` | No code change — named as the concrete out-of-taxonomy instance |

Explicitly **not** touched: `api-specification/` (N1); `cobra.ExactArgs(1)`/`Parse`'s guard/main's
usage-error path (C1); `Deps` (N4); no `adapter_test.go` is created (C0, §4 total is 51 with **zero**
adapter/head/I-O rows). Full table incl. non-module test surfaces: [`change-delta.md`](./change-delta.md) §2.

### 2.5 RED → GREEN scenarios (the coverage this change buys)

| Ref | Where | Fault it discriminates | Current suite verdict | Changed suite verdict |
|---|---|---|---|---|
| **A** (component scenario 9) | `validate.feature`, `@wip` | `adapter.go` primary-verdict arm (`case err==nil: return 1`) silently deleted/changed | **GREEN** (blind spot — nothing asserts exit 1) | **RED→GREEN**: exit 1, schema-valid canon-1.1, `compatible==false`, `errors[0].code=="CHANNEL_NOT_IN_PROVIDER"` |
| **B** (`main_test.go`, C4) | out-of-taxonomy fallback | `ReportWriter.Write` failure wrapped as a sentinel, or `ResolveExitCode`'s totality narrowed | **GREEN** (blind spot) | **RED→GREEN**: `*exitError{code:3}`; stdout body deliberately **not** asserted (schema-invalid today — recorded debt, not fixed here) |
| **C** (`main_test.go`, C1, 2 cases) | argc≠1 guard | both `cobra.ExactArgs(1)` and `Parse`'s own guard dropped simultaneously | **GREEN** (blind spot) | **RED→GREEN**: non-nil error, **not** an `*exitError`, for argc=0 and argc=2 |
| **D** (I2, new Then-step on scenario 1) | happy scenario | any byte-level drift in stdout report bytes (indentation, key order, dropped field) | **GREEN** (existing step too insensitive — checks only `compatible`/`errors` length) | **RED→GREEN**: full-byte diff against `report.baseline.json` (only `generated_at`'s value normalized) exits 0 |
| **E** (I1c, new Go test + golden) | `internal/validate` | `Reporter` reading the clock twice, or `generated_at`'s layout changing | **GREEN** (existing fake clock returns a fixed instant — undetectable) | **RED→GREEN**: counting fake asserts exactly 1 `Now()` call; full-byte golden comparison incl. `generated_at` |

Grid closure (C2): after row A the 7-row `cli.ResolveExitCode` grid is proven **7/7 through the
binary with zero new unit tests** — row 2 (exit 1) was the single gap. Component-scenario gate
becomes `N = 2 (happy-class) + 7 (adapter branches) = 9`. Full discriminating table:
[`change-delta.md`](./change-delta.md) §3; grid matrix + scenario table: [`contracts.md`](./contracts.md) §5/§6.

**Anti-gaming boundary:** row B MUST NOT become a 10th component scenario — a report-write failure is
not one of the 7 adapter branches. It lives in `cmd/app/main_test.go` for exactly that reason.

### 2.6 Failure-mode map (unchanged — regression envelope, every row identical before/after)

| `error.code` | Raised by (post-refactor name) | Exit | In `errors[]`? | Report on stdout? |
|---|---|---|---|---|
| `CONFIG_ERROR` | `ConfigStore.Load`, `NewConfig`, `NewComparison(ComparisonInput)` | 2 | no | no |
| `FILE_NOT_FOUND` | `ContractStore.Load`, `FileSpecLoader.Load` | 3 | yes | yes (synthesized) |
| `PARSE_ERROR` | `ContractParser.Parse`, `File`/`HTTPSpecLoader.Load` | 3 | yes | yes (synthesized) |
| `HTTP_ERROR` | `HTTPSpecLoader.Load` | 3 | yes | yes (synthesized) |
| `TIMEOUT_ERROR` | `HTTPSpecLoader.Load` | 3 | yes | yes (synthesized) |
| `CHANNEL_NOT_IN_PROVIDER`, `PROTOCOL_MISMATCH`, `DIRECTION_NOT_IN_PROVIDER`, `MESSAGE_NOT_IN_PROVIDER` (R1–R4) | `ResolveChannelDirection` | **1** | yes | yes (canon-1.1) |
| `MISSING_REQUIRED_SENT_FIELD`, `READS_FIELD_NOT_PROVIDED`, `TYPE_MISMATCH`, `CONTENT_TYPE_MISMATCH`, `CORRELATION_ID_MISMATCH` (R5–R9) | `CompareMessage` | **1** | yes | yes (canon-1.1) |
| *(none — out of taxonomy)* | `ReportWriter.Write` disk/marshal failure | **3** (total-function fallback, C4/D3) | n/a — synthesized `code:""`, schema-invalid; **not asserted, not fixed here** (debt) | yes (synthesized, non-canon) |

Client/operator action per exit code: **0** — proceed, consumer compatible; **1** — CI MUST fail the
gate, the consumer's declared usage is incompatible with the provider's actual spec (`errors[]`
names each violated rule); **2** — operator/config-author error, fix `config.yaml`/the consumed
contract, no report was produced; **3** — I/O or parse fault (missing file, unparseable spec,
provider unreachable/timeout, or an unclassified fault via the `default:` fallback) — retry or fix
connectivity/artifact availability, `errors[0].code` (when present) names the cause. Source:
BRD §6 (`.agent/planner/brd.md`), restated for traceability in `contracts.md` §2.

### 2.7 Ticket list (7, dependency-ordered — sequencing is load-bearing, not incidental)

| # | Type | What it does | Outputs | `blocked_by` |
|---|---|---|---|---|
| 01 | component (RED-first) | Scenario 9 (exit-1 verdict) + `incompatible/` fixture + I2 stdout baseline. **Runs first, on unmodified code (HEAD=`9eef9e9`)** — captures the byte anchor that would be voided by any prior edit. | `validate.feature`, `validate_steps.go`, `fixtures/validate/incompatible/*`, `fixtures/validate/good/report.baseline.json` | `[]` |
| 02 | module (D10 determinism anchor) | `ProcessValidate` golden test under a frozen, counting `Clock` (I1c/I1a). **Also runs on unmodified production code** — the second, and last, byte anchor; blocks every reshape ticket (04/05/06) directly for that reason. | `internal/validate/processvalidate_determinism_test.go`, `testdata/report.golden.json` | `[01]` |
| 03 | module (`main_test.go` +3 cases) | Pins the two previously-uncovered branches: C1 argc-breach short-circuit (×2, argc=0/argc=2) + C4 out-of-taxonomy fallback (×1). Still runs **before** the reshape — regression envelope, not after it. Adds 0 grid rows (row 2 already closed by ticket 01). | `cmd/app/main_test.go` | `[01, 02]` |
| 04 | module | `BuildContractParser` + `ContractParser.Parse` — one reshape ticket (factory + product method cannot compile apart, ADR-001). | `logic.go`, `head.go`, `newconsumedcontract_test.go`, `buildcontractparser_test.go` | `[01, 02, 03]` |
| 05 | module | `NewComparison`/`ComparisonInput` — materializes the three-stream join as a named type. | `domain.go`, `logic.go`, `head.go`, `newcomparison_test.go` | `[02, 04]` |
| 06 | module | `BuildReporter` + `Reporter.Fold` — encloses the clock port. **`blocked_by` names ticket 02 directly** (not only via the chain) — this is the node the byte anchors exist to protect; a golden mismatch after this edit is a real defect, not a stale artifact → STOP, don't regenerate. | `logic.go`, `head.go`, `foldreport_test.go`, `buildreporter_test.go` | `[02, 05]` |
| 07 | module (sink — head rewiring) | `ProcessValidate` reads as one bind block + a chain where every step takes exactly one data entity. Signature/`Deps` unchanged (BRD N4) — out of lane to touch, → STOP if attempted. | `head.go` | `[06]` |

Execution order is linear and fully determined by `blocked_by`: **01 → 02 → 03 → 04 → 05 → 06 → 07**.
Full ticket bodies: [`tickets/ticket-01.md`](./tickets/ticket-01.md) … [`ticket-07.md`](./tickets/ticket-07.md).

### 2.8 Regression invariants (MUST hold before AND after, whole change)

- **I2** — for the `good/` fixture, post-refactor stdout bytes equal the pre-refactor baseline
  byte-for-byte (only `generated_at`'s *value* normalized).
- **I1 (a/b/c)** — exactly one `Now()` call per `Fold`; `time.Now()` appears only in `register.go`;
  the `ProcessValidate` golden matches full bytes incl. `generated_at`.
- **N2** — all 8 pre-existing component fixtures: identical exit code to the `9eef9e9` baseline.
- **N5** — 0 assertions deleted/weakened in `foldreport_test.go`, `newconsumedcontract_test.go`,
  `newcomparison_test.go` (call-site updates only).
- **N7 (as amended by ADR-001)** — `contracts.md` §4's rule sentence unchanged; total is **51**
  (was BRD's stale 49), with exactly the two new branchless factory rows plus the two renames.
- **I3** — `go build/vet/test ./...` and `component-tests/scripts/run-tests.sh` exit 0, **9/9**
  validate scenarios + smoke green, scenario 9 stays `@wip` until `@fagan` acceptance.
- **Escalation trigger (E1)** — if any of the 8 pre-existing fixtures yields a different exit code,
  or `good/` a different report byte (modulo `generated_at`'s value), the `patch` assumption is
  broken → STOP and re-triage (`change-delta.md` §6).

### 2.9 Open questions / tech debt

- **0 open questions** — BRD is agent-ready (`.agent/planner/brd.md`, all 6 draft questions
  resolved by the operator 2026-07-26, §10).
- **New debt recorded, not fixed here** (BRD C4 follow-up, `change-delta.md` row B / `contracts.md`
  §`ReportWriter.Write`): the out-of-taxonomy exit-3 path's synthesized stdout body carries
  `errors[0].code == ""` and no `subject` — schema-invalid against `report.schema.json` (13-member
  enum, `subject` required). Pinning or fixing it is a behaviour change outside the `patch` envelope;
  needs a frozen-contract decision in a later ticket.
- **Design correction carried forward**: this change's design package supersedes BRD N7/D1's pinned
  unit-test total of 49 with **51** (ADR-001). The BRD itself is not edited (it is a frozen input);
  the correction is recorded here and in `contracts.md` §4 for the operator to accept knowingly at
  this gate.

### 2.10 What to check at the gate

- Both byte anchors (I2 baseline, I1c golden) captured from `9eef9e9` **before** ticket 04/05/06
  touch any signature — ticket 01/02 ordering enforces this; verify `blocked_by` graph matches §2.7.
- ADR-001's node-count consequence (19→21, 49→51) is accepted as a correction to BRD N7/D1, not
  silently overridden.
- No `internal/validate/adapter_test.go` exists; no direct unit test calls the **CLI ingress
  adapter's** package-level `cli.Parse` / `cli.ResolveExitCode` — in Go, the package-level
  `validate.Parse(` / `validate.ResolveExitCode(` of `internal/validate/adapter.go`
  (C0, mechanical fit criterion). Mechanically:

  ```sh
  grep -nE '(^|[^.[:alnum:]_])(Parse|ResolveExitCode)\(' internal/validate/*_test.go   # must be empty
  ```

  The leading `[^.]` guard is the whole point: a **receiver-qualified** `.Parse(` is a *different*
  symbol and is **not** in C0's scope. Specifically `ContractParser.Parse(RawContract)` — the domain
  method this change introduces (`contracts.md` §`ContractParser.Parse`, ADR-001, ticket-04) — is
  domain logic, not the ingress adapter; its direct unit tests (`newconsumedcontract_test.go`,
  `buildcontractparser_test.go`; `contracts.md` §4 rows 7–8) are **mandated**, not a C0 violation.
  A bare `Parse(` is the adapter; a `.Parse(` on a `ContractParser` is not.
- Component-scenario gate `#component_adapter_failure_scenarios (7) == #distinguishable_adapter_branches (7)`
  holds after scenario 9 is added; total scenarios = 9, not 10 (anti-gaming boundary, §2.5).
- `api-specification/` diff = 0 files (N1); `go.mod`/`component-tests/go.mod` diff = 0 added
  `require` lines (N3).

## 3. Next step

Operator reviews this PLAN.md (+ linked artifacts) at **Gate #1**. On accept → `implementer` works
tickets 01–07 in the fixed order above (isolated worktree per harness discipline; tickets 01/02 run
on unmodified code and must land before any reshape ticket touches a signature). On reject/changes →
back to `planner`/`plan-reviewer` per the fix-loop rules in the harness `CLAUDE.md`.

<!-- DONE: PLAN 001-pipe-arity-coverage-gaps -->
