# change-delta — 001-pipe-arity-coverage-gaps

> **Stage: change-intake (Wirth).** Lane `patch` (`.agent/planner/mode`), target `cli`
> (`.agent/planner/target`). In: [`.agent/planner/brd.md`](../../../../../.agent/planner/brd.md)
> (agent-ready, 0 open questions; source BR [`debt/task-001.md`](../../../../../debt/task-001.md)).
> Slice: `slice-01-validate` — existing design package
> [`module-tree.md`](../../module-tree.md) · [`contracts.md`](../../contracts.md) · [`c4.md`](../../c4.md).
> Baseline commit: **`9eef9e9`** (= current `HEAD`; the I2 baseline and the I1c golden are capturable
> right now, before any edit).
> **Design signal: `needed`** — rationale in §5.

## 1. Change statement + rationale

Three nodes of the `slice-01-validate` pipe take **two** data arguments (`FoldReport(outcome, clock)`,
`NewConsumedContract(raw, expectedConsumer)`, `NewComparison(cfg, contract, pchans)`), and two
behavioural branches of the CLI are asserted by **no test anywhere** — the primary verdict exit `1`
and the out-of-taxonomy fallback exit `3`. This change rebinds the three call sites to the
already-established bind-then-chain idiom (`BuildReporter(clock).Fold(Outcome)`,
`BuildContractParser(consumerName).Parse(RawContract)`, `NewComparison(ComparisonInput{...})` — one
*named* join for the one genuine three-stream union), and closes both coverage holes.

The load-bearing reason is **falsifiability, not tidiness**. The arity edit is a pure restructure:
by construction it changes no observable output, so on its own it is *unfalsifiable* — no test can
tell the refactor apart from a botched refactor. What makes it a safe `patch` is the pair of
byte-level anchors captured from the pre-refactor binary (I2 stdout baseline, I1c `ProcessValidate`
golden under a frozen `Clock`), and what makes it *worth* doing now is that the same pass buys back
the two grid rows the suite could never see. Both anchors MUST be captured from `9eef9e9` **before**
the first signature is touched; regenerating either from post-refactor output voids the proof and
turns this into an unverified change.

Public surface is untouched: `api-specification/*.schema.json` frozen (N1), argv/exit-code grid/report
bytes identical (N2/I2). `internal/validate` is a Go `internal/` package — no external importer exists,
so the three signature changes are not a compatibility break. **`patch` holds; §4 spec-delta is N/A.**

## 2. Affected modules

All rows are existing nodes of [`module-tree.md`](../../module-tree.md) §3 in package
`internal/validate/` (`io: none` throughout — this change adds no port, touches no `io:` tag, and
introduces **no new module**; `HTTPSpecLoader.Load`, the slice's only `io: http` node, is untouched).

| # (tree) | Module | File | `io:` | Nature of the edit |
|---|---|---|---|---|
| 17 | `FoldReport` → `BuildReporter` + `Reporter.Fold` | `logic.go:1074` | `none` | **Reshape.** The `Clock` port moves from a call argument to the collaborator's construction; `Fold(Outcome) Report` keeps one data input. Hidden decision (canon-1.1 shape + where `generated_at` comes from) unchanged. D10 tightens: the single clock read now lives *inside* `Reporter`. |
| 7 | `NewConsumedContract` → `BuildContractParser` + `ContractParser.Parse` | `logic.go:120` | `none` | **Reshape.** `expectedConsumer` — an already-`NewConfig`-validated config scalar — moves to construction; `Parse(RawContract)` keeps one data input. Antecedent set and `ErrParseError`→`PARSE_ERROR`→exit 3 unchanged verbatim. |
| 13 | `NewComparison` | `logic.go:576` | `none` | **Join named.** Three streams genuinely unite here, so the arity is not removed — it is materialized as `ComparisonInput{Config, Contract, ProviderChannels}`. The cross-artifact check (every `cfg.consumer.channels` address present in `contract.Channels` → `ErrConfigInvalid` → exit 2) is preserved verbatim. |
| 2 | `ProcessValidate` (head) | `head.go:35` | `none` | **Rewiring only.** Becomes bind-then-chain: `reporter`/`parser` bound in the pre-chain block alongside the existing `loader`/`writer`; zero steps take two data arguments. Its own signature `(Invocation, Deps) (Report, error)` and `Deps{Clock, HTTPClient}` are **unchanged** — I1c depends on the head being callable identically before and after. |
| 3 | `cli.ResolveExitCode` | `adapter.go:32` | `none` | **No code change.** Documentation only (D3: totality over any `error`, `default:` as the out-of-taxonomy fallback = 3) + it becomes the module the new coverage rows §3-A/§3-B actually discriminate. |
| 19 | `ReportWriter.Write` | `io_report.go:42` | `none` | **No code change.** Named in D3 as the concrete reachable instance of an error wrapping no sentinel; its `contracts.md` "out of scope" note stays. |

Non-module surfaces in the blast radius (not tree nodes; listed so the ticketer misses none):

| Surface | Path | Edit |
|---|---|---|
| Existing unit tests | `foldreport_test.go`, `newconsumedcontract_test.go`, `newcomparison_test.go` | Mechanical call-site updates only. **0** assertions deleted or weakened (N5). |
| Component feature | `component-tests/features/validate.feature` | +1 scenario (`@wip`), +1 Then-step on the happy scenario, header formula/STOP-warning restated (D2). |
| Component fixture | `component-tests/fixtures/validate/incompatible/` | New dir, minimal mutation of `good/`. `good/`'s three files MUST NOT be modified (I2 depends on their bytes). |
| Baseline artifact | `component-tests/fixtures/validate/good/report.baseline.json` | New; captured from the `9eef9e9` binary. Assertion-only artifact. |
| Golden artifact | `internal/validate/testdata/report.golden.json` | New; captured from pre-refactor `ProcessValidate` under a frozen `Clock`. |
| Binary-level tests | `cmd/app/main_test.go` | +3 cases (C1 ×2 argc-breach, C4 ×1 out-of-taxonomy). |
| Design package | `module-tree.md` §3/§4, `contracts.md` §1/§3/§4/§5/§6, `c4.md` C3 | See §5 — this is the ripple that makes `design=needed`. |

**Explicitly NOT touched:** `api-specification/` (N1); `cobra.ExactArgs(1)`, `Parse`'s guard, main's
usage-error path (C1 MUST NOT); `Deps` (N4); `internal/validate/adapter_test.go` MUST NOT be created
(C0 — the frozen "adapters are not unit-tested" rule stands verbatim; §4's total was written here as
49 — **superseded by ADR-001 → 51**, see `contracts.md` §4 / `module-tree.md`; C0 itself is unchanged).

## 3. Affected component scenarios — discriminating analysis

**The arity refactor (R1–R5) owns ZERO rows in this table, by design.** It is a pure restructure: for
every one of the 8 existing fixtures the binary's exit code and report bytes are identical before and
after, so naming any existing scenario as "changing" would be a fabrication. The entire current
suite — 49 unit tests + 8 component scenarios + smoke — is therefore the **invariant to keep green**
(I3), and the two byte anchors (rows D/E) are what convert "behaviour-preserving" from an assertion
into a measurement.

The rows below are the *coverage* delta. Because the production behaviour is unchanged on these paths
too, "output(current) vs output(changed)" is evaluated on the axis that actually differs: **the
suite's verdict on a named fault**. Each row states the concrete fault, then the counterfactual
verdict of the current suite and of the changed suite on that same fault. Rows whose verdicts matched
would be degenerate; none do.

| # | Scenario (where it lives) | Input | Fault it must discriminate | Verdict (current suite) | Verdict (changed suite) | RED-reason |
|---|---|---|---|---|---|---|
| **A** | **C3 — new component scenario 9**, `validate.feature` | `fixtures/validate/incompatible/config.yaml`: `good/` + one extra channel address present in **both** `consumer.channels` and the consumed-contract's `channels`, absent from the provider spec → rule R1 | `adapter.go:39` `case err == nil: return 1` → `return 0` (or the arm deleted) | **GREEN** — nothing asserts exit 1 through the binary. `contracts.md` §4 has no `ResolveExitCode` row (adapters are not unit-tested) and all 8 scenarios assert 0/2/3. The tool's *primary verdict* would silently become "always compatible". | **RED** — scenario 9 asserts exit **1**, then schema-valid canon-1.1 stdout, `compatible == false`, `errors[0].code == "CHANNEL_NOT_IN_PROVIDER"` | Scenario and fixture do not exist → RED by absence (business reason, `component-tests` REALIZE half). Carries `@wip`; the 8 existing scenarios stay untagged. |
| **B** | **C4 — new case**, `cmd/app/main_test.go` | config with a reachable, compatible pair + `save_json_report: true` + `json_report_file` inside a **non-existent directory** → `os.WriteFile` fails (`io_report.go:53`), error wraps **no** sentinel | `io_report.go:53` "helpfully" wraps the write failure as `%w: ErrConfigInvalid` (or `ResolveExitCode` gains an arm narrowing its totality) | **GREEN** — no fixture ever reaches `io_report.go`'s error return; the out-of-taxonomy path is executed by nothing. The generic `default: return 3` *is* exercised by scenarios 3a–4d, so mutating that literal is caught — but the *totality* claim is not. | **RED** — the case asserts `*exitError` with `code == 3`; under the fault it observes 2 | Case does not exist → RED by absence. **MUST assert nothing else**: today's stdout body on this path carries `code: ""` and no `subject` — schema-**invalid** against `report.schema.json`. Pinning or fixing it exceeds the `patch` envelope (recorded as new debt, N1). |
| **C** | **C1 — 2 new cases**, `cmd/app/main_test.go` | `root.Execute()` with `validate` + **0** args, and with **2** args | both guards dropped at once: `Args: cobra.ArbitraryArgs` **and** `Parse` reading `args[0]` unguarded — i.e. the invariant "argc ≠ 1 never reaches the pipe" is lost | **GREEN** — every component scenario passes exactly one positional arg; argc ≠ 1 is exercised by nothing. | **RED** — 2 args would reach the pipe on `args[0]` and return `*exitError{code:2}`; 0 args would panic. Both cases assert a non-nil error that is **not** an `*exitError`. | Cases do not exist → RED by absence. Defense-in-depth is intended: dropping *one* guard keeps the invariant and the cases correctly stay green — that is the invariant holding, not insensitivity. |
| **D** | **I2 — new Then-step on the EXISTING happy scenario** (scenario 1) | `fixtures/validate/good/config.yaml`, unchanged bytes | any byte-level drift in the stdout report that survives a JSON decode — e.g. `report.WriteJSON` switching `json.MarshalIndent(v, "", "  ")` → `json.Marshal` (compact), a reordered/renamed constant, a changed `generated_at` layout | **GREEN** — **this is the test-input rework this delta owes.** The existing step decodes stdout into a struct and checks only `compatible == true` and `len(errors) == 0` (`validate_steps.go:102`); the sibling step checks only `len(uncovered_channels) != 0`. Indentation, key order, `schema_version`, `validator`, `interaction`, `provenance` echo and `generated_at` are all invisible to it. The current happy scenario is **too insensitive to prove a behaviour-preserving refactor**. | **RED** — `diff` against `report.baseline.json` (captured from the `9eef9e9` binary) exits non-zero. Normalization is surgical: `generated_at` must be **present** and parse as **RFC3339**, and only its *value* is replaced identically on both sides; every remaining byte must be equal. | Baseline artifact does not exist → capture from `9eef9e9` first, then the step is RED until the post-refactor bytes match. **No scenario inflation** — this is a Then-step, so the §6 count stays 9. |
| **E** | **I1c + I1a — new Go test**, `internal/validate/` (+ `testdata/report.golden.json`) | `ProcessValidate(inv, Deps{Clock: frozenClock, HTTPClient: …})` over the `good/` pair, config assembled at run time over those same files, `save_json_report: false` | `Reporter` reading the clock **twice** (or per-violation), or `generated_at`'s layout changing | **GREEN** — `foldreport_test.go` hands in a `fixedClock` that returns the *same* instant on every call and compares against a `want` built from that same instant, so a second `Now()` is undetectable; a layout change would fail that one test only if the literal `"2026-07-25T12:00:00Z"` is also touched, and nothing counts calls. | **RED** — a counting fake asserts exactly **1** `Now()` call across one `Fold`, and the golden comparison covers the **full** bytes **including** `generated_at`. | Test and golden do not exist. Authored and captured against **pre-refactor** code and MUST be green **both before and after** (`ProcessValidate`'s signature is unchanged by R1–R5). MUST NOT fork `good/`'s contract/spec bytes — it reads the same files, so I2 and I1c cannot drift apart. Doc comment MUST mark it a **D10 determinism regression anchor, not a head unit test** (precedent: `report_json_shape_test.go`); §4's total was written here as 49 — **superseded by ADR-001 → 51**. |

**Grid closure (C2), mechanically checkable.** After row A the 7-row `ResolveExitCode` grid is proven
**7/7 through the binary** with **zero** new unit tests: row 1 (exit 0) ← scenario 1; row 2 (exit 1)
← **new scenario 9**; row 3 (`ErrConfigInvalid` → 2) ← scenario 2a + `TestValidateCmd_ConfigFileNotFound`;
rows 4–7 (`FILE_NOT_FOUND` ×2, `PARSE_ERROR` ×2, `HTTP_ERROR`, `TIMEOUT_ERROR` → 3) ← scenarios
3a/4a, 3b/4b, 4c, 4d. Every error is wrapped exactly as production raises it **by construction** —
the binary path *is* production. Row 2 was the single gap.

**Anti-gaming boundary.** Row B MUST NOT become a 10th component scenario: a report-write failure is
**not** one of the 7 adapter branches and would break the §6 gate the same change is restating. It
lives in `main_test.go` precisely for that reason.

## 4. Spec-delta

**N/A — lane is `patch`.** `api-specification/{config,consumed-contract,report}.schema.json` are frozen
and untouched; the fit criterion is `git diff --stat api-specification/` = **0 files changed** (N1).
No operation, field, or error code is added, removed, re-typed, or newly required. The CLI's argv
surface, exit-code grid and report bytes are all inside the regression envelope (§3 rows D/E).

The design-package documents *do* change (below) — those are the harness's internal design record,
not the frozen machine contract, and editing them on a `patch` is in-lane.

## 5. Design signal: `needed`

Three independent ripples, each an interrelation several artifacts must agree on, each with a real
competing approach. They must be pinned **once**, before tickets are cut — otherwise each ticket
resolves them differently.

1. **One node or two?** `module-tree.md` §3's existing convention splits a factory from its product's
   method into **separate** rows — `BuildSpecLoader` (8) / `FileSpecLoader.Load` (9) / `HTTPSpecLoader.Load` (10),
   and `BuildReportWriter` (18) / `ReportWriter.Write` (19) — and both factories carry their own
   `contracts.md` §4 rows (N=2 each). Applying that convention to `BuildReporter`/`Reporter.Fold` and
   `BuildContractParser`/`ContractParser.Parse` takes the tree **19 → 21** and §4's total **49 → 51**.
   BRD **N7/D1** binds the opposite: §4 keeps exactly **49** with the two rows merely *renamed*
   (`FoldReport` → `Reporter.Fold`, `NewConsumedContract` → `ContractParser.Parse`), i.e. one node each.
   Both readings are defensible; the delta cannot choose without redesigning the tree, which is not
   this stage's authority. **This is the decision the design stage must settle**, and it propagates to
   §3's table + its "19 modules" count sentence, §4's rows and total, §3's Notes (whose "three
   documented, sanctioned exceptions" list must end up **empty** — D1), and `c4.md`, whose C3 *is* the
   module tree (`Component(newctr…)`, `Component(fold…)` and the `Rel(head, …)` labels at c4.md:48–83
   name the old signatures verbatim).
2. **A frozen gate is being amended.** `contracts.md` §6's formula moves from `N = 1 (happy) + 7 = 8`
   to `N = 2 (happy-class: compatible + incompatible verdict) + 7 (adapter branches) = 9`, the gate
   line to `#component_adapter_failure_scenarios (7) == #distinguishable_adapter_branches (7)`, the
   scenario table gains row 9, and `validate.feature`'s header STOP-warning shifts from "a 9th
   scenario" to "a **10th**". The feature file's own header declares this a **design act, not a
   realization act** ("STOP, back to wirth-moduledesigner / contracts.md") — the harness's own rule
   routes this through design.
3. **§5 gains rows the ticketer cannot invent.** The Gherkin↔module reconciliation table needs one row
   per new distinguishable Then-step — the exit-1 verdict (`CompareContracts` → `Reporter.Fold` →
   `cli.ResolveExitCode`), `errors[0].code == CHANNEL_NOT_IN_PROVIDER` (`ResolveChannelDirection`, R1),
   and the I2 baseline-bytes step on the happy scenario — plus the C2 7-row grid matrix (D2) and the
   `cli.ResolveExitCode` totality card (D3).

Ordering constraint the design stage MUST carry into the plan: **capture both anchors from `9eef9e9`
before any signature is edited** (§3 rows D/E). It is the one sequencing decision that cannot be
recovered later.

## 6. Fitness check

| Rule | Verdict |
|---|---|
| Design package exists (`docs/design/slice-01-validate/`) | PASS — module-tree, contracts, c4, use-case, PLAN all present |
| `patch` without a contract change | PASS — `api-specification/` frozen and untouched (N1); design-package edits are not the machine contract |
| Not a break for any consumer | PASS — `internal/` package, no external importer; argv/exit grid/report bytes inside the regression envelope |
| Delta of existing code, not a new slice | PASS — one existing slice, one package, no new external input, no new module |
| Ripple radius | `design=needed` (§5) |

**Escalation trigger (E1, from the BRD):** if any of the 8 pre-existing fixtures yields a different
exit code, or `good/` a different report byte (modulo `generated_at`'s value), the `patch` assumption
is broken → STOP and re-triage.

<!-- DONE: change-delta 001-pipe-arity-coverage-gaps -->
