# PLAN — slice-01-validate

> Plan-index for Gate #1 (operator acceptance). Assembled by `wirth-planner` from the finished
> design package below. This file designs nothing — it indexes paths and summarizes for the
> operator's approve/reject decision.

## 1. Design package (path index)

| Artifact | Path |
|---|---|
| Cockburn use case (UC1) | [`use-case.md`](./use-case.md) |
| Module tree + head-pipe pseudocode | [`module-tree.md`](./module-tree.md) |
| Module contracts + unit-test formula + component-scenario set | [`contracts.md`](./contracts.md) |
| C4 (C2 Container + C3 Component) | [`c4.md`](./c4.md) |
| Tickets (23, dependency-ordered) | [`tickets/`](./tickets/) |
| Config contract (frozen) | [`../../../api-specification/config.schema.json`](../../../api-specification/config.schema.json) |
| Consumed-contract contract (frozen) | [`../../../api-specification/consumed-contract.schema.json`](../../../api-specification/consumed-contract.schema.json) |
| Report contract (frozen) | [`../../../api-specification/report.schema.json`](../../../api-specification/report.schema.json) |
| Repo README (documentation) | [`../../../README.md`](../../../README.md) |
| BRD / FRD / slice backlog | [`../../../.agent/planner/brd.md`](../../../.agent/planner/brd.md), [`frd.md`](../../../.agent/planner/frd.md), [`slices.md`](../../../.agent/planner/slices.md) |
| Route classification | [`../../../.agent/planner/mode`](../../../.agent/planner/mode) |

Package scope: single slice (`slice-01-validate`, package `internal/validate/`), single external
input (one-shot CLI invocation `pinout-asyncapi validate <config.yaml>`), single Cockburn use case
(UC1). No epic-level decomposition — modular level, per-slice chain
`planner → plan-reviewer → [Gate #1] → implementer → fixer`.

## 2. Gate #1 summary

### 2.1 What this slice does

`pinout-asyncapi validate <config.yaml>` checks whether a consumer's declared AsyncAPI usage
(`consumed-contract`) is compatible with a provider's actual AsyncAPI 3.0 spec (`spec_path` local
file XOR `spec_url` over HTTPS with a bearer token from env), and emits one canon-`1.1` JSON report
+ one exit code (0/1/2/3). Read-only against the provider: its spec is ground truth, never mutated
or called. Primary actor: consumer CI pre-merge gate (also runnable by a developer locally).

### 2.2 Head-pipe functional block (linear ROP pipe, `ProcessValidate`)

```
cli.Parse(args) -> Invocation
  -> ConfigStore.Load(ConfigPath)                 -> RawConfig          [fs]
  -> NewConfig(raw)                                -> Config             [validate]
  -> ContractStore.Load(consumed_contract_path)    -> RawContract        [fs]
  -> NewConsumedContract(raw, consumer.name)       -> ConsumedContract   [validate]
  -> BuildSpecLoader(provider, timeout)            -> SpecLoader         [factory: file XOR http]
  -> SpecLoader.Load()                             -> ProviderSpec       [fs XOR http; InlineServerRefs -> lib.Process()]
  -> DeriveProviderChannels(spec)                  -> ProviderChannels   [projection: no-messages / reply defaults]
  -> NewComparison(cfg, contract, pchans)          -> Comparison         [unite + channel-scope cross-check]
  -> CompareContracts(comparison)                  -> Outcome            [CORE: R1-R9 fold, no short-circuit across channels]
  -> FoldReport(outcome)                           -> Report             [canon 1.1; clock.Now() called exactly once]
  -> BuildReportWriter(settings) -> ReportWriter.Write(report)           [iff save_json_report]
  -> Ok(report)
-> cli.ResolveExitCode(result) -> exit 0/1/2/3; report always -> stdout; logs -> stderr
```

19 modules total (18 `io: none` + 1 `io: http` — `HTTPSpecLoader.Load`). No `db`/`llm`/`queue`
touchpoints. Full tree, contracts, and Mermaid C4 diagrams: [`module-tree.md`](./module-tree.md) §3-4,
[`c4.md`](./c4.md).

### 2.3 Failure-mode map (exit-code grid)

| Exit | Class | `error.code`(s) | Report written? | Source |
|---|---|---|---|---|
| 0 | compatible | — (`errors: []`) | yes | `CompareContracts` finds no violations |
| 1 | incompatible (domain verdict, not a tool fault) | `CHANNEL_NOT_IN_PROVIDER` (R1), `PROTOCOL_MISMATCH` (R2), `DIRECTION_NOT_IN_PROVIDER` (R3), `MESSAGE_NOT_IN_PROVIDER` (R4), `MISSING_REQUIRED_SENT_FIELD` (R5), `READS_FIELD_NOT_PROVIDED` (R6), `TYPE_MISMATCH` (R7), `CONTENT_TYPE_MISMATCH` (R8), `CORRELATION_ID_MISMATCH` (R9) | yes | `ResolveChannelDirection` / `CompareMessage` |
| 2 | config | `CONFIG_ERROR` | **no** | `ConfigStore.Load` (unreadable), `NewConfig` (schema breach), `NewComparison` (channel scope not in consumed-contract) — 3 raise-sites, 1 code, detected before any report exists |
| 3 | io · parse | `FILE_NOT_FOUND`, `PARSE_ERROR`, `HTTP_ERROR`, `TIMEOUT_ERROR` | yes (`errors[]` carries the code) | `ContractStore.Load`, `FileSpecLoader.Load`/`HTTPSpecLoader.Load` |

16 use-case Extensions == 16 failure-mode rows == 14 distinct `error.code` values (`CONFIG_ERROR` a
14th state, outside `errors[]`). `PINOUT_PROVIDER_TOKEN` never appears in report or logs on any
branch. Full row-by-row detail: [`use-case.md`](./use-case.md) Extensions, [`contracts.md`](./contracts.md) §2.

Test coverage split (by kind, not by letter): the 9 verdict rules R1-R9 (Extensions 6a-6i) are
**unit-tested** (49 unit tests total, formula in `contracts.md` §4) as pure logic; the 7
distinguishable I/O-adapter failure branches (`CONFIG_ERROR`×1, `FILE_NOT_FOUND`×2, `PARSE_ERROR`×2,
`HTTP_ERROR`×1, `TIMEOUT_ERROR`×1) + 1 happy path = **8 component scenarios**
(`contracts.md` §6, ticket-02, `@wip` until realized).

### 2.4 Ticket list (23, dependency-ordered)

| # | Type | Outputs | Blocked by |
|---|---|---|---|
| 01 | scaffold | `go.mod`, `cmd/app/main.go`, shared config/report shells, component-test harness | — (blocks all) |
| 02 | component (RED) | `component-tests/features/validate.feature` + steps | 01 |
| 03 | module (foundation) | `internal/validate/domain.go`, `errors.go` | 01, 02 |
| 04 | module | `adapter.go` (`cli.Parse`) | 01-03 |
| 05 | module | `adapter.go` (`cli.ResolveExitCode`) | 01-04 |
| 06 | module | `io_config.go` (`ConfigStore.Load`) | 01-03 |
| 07 | module | `logic.go` + test (`NewConfig`) | 01-03 |
| 08 | module | `io_contract.go` (`ContractStore.Load`) | 01-03 |
| 09 | module | `logic.go` + test (`NewConsumedContract`) | 01-03 |
| 10 | module | `logic.go` + test (`InlineServerRefs`) | 01-03 |
| 11 | module | `io_spec.go` (`FileSpecLoader.Load`) | 01-03, 10 |
| 12 | module (`io: http`) | `io_spec.go` (`HTTPSpecLoader.Load`) | 01-03, 10, 11 |
| 13 | module | `logic.go` + test (`BuildSpecLoader`) | 01-03, 11, 12 |
| 14 | module | `logic.go` + test (`DeriveProviderChannels`) | 01-03 |
| 15 | module | `logic.go` + test (`NewComparison`) | 01-03, 07, 09, 14 |
| 16 | module | `logic.go` + test (`ResolveChannelDirection`, R1-R4) | 01-03, 15 |
| 17 | module | `logic.go` + test (`CompareMessage`, R5-R9) | 01-03, 15 |
| 18 | module | `logic.go` + test (`CompareContracts`) | 01-03, 15-17 |
| 19 | module | `logic.go` + test (`FoldReport`) | 01-03, 18 |
| 20 | module | `io_report.go` (`ReportWriter.Write`) | 01-03 |
| 21 | module | `logic.go` + test (`BuildReportWriter`) | 01-03, 20 |
| 22 | module (head composition) | `head.go` (`ProcessValidate`) | 01-03, 06-09, 11, 13-15, 18-21 |
| 23 | module (wiring) | `register.go`, `cmd/app/main.go` | all (01-22) |

Structure: 1 scaffold (blocks all) + 1 component ticket (RED, drives the `@wip` feature file into
place) + 1 foundation ticket (domain/errors) + 18 leaf-first module tickets (I/O objects and pure
logic, parallelizable within their dependency tier) + 1 head-composition ticket + 1 final wiring
ticket. Full ticket bodies: [`tickets/ticket-01.md`](./tickets/ticket-01.md) … [`ticket-23.md`](./tickets/ticket-23.md).

### 2.5 What to check at the gate

- Contracts frozen and consistent: `config.schema.json`, `consumed-contract.schema.json`,
  `report.schema.json` (all present in `api-specification/`).
- Module tree = C3 node-for-node (`module-tree.md` §3 ≡ `c4.md` C3); one data argument per node with
  3 documented, sanctioned exceptions (`NewComparison`, `NewConsumedContract`, `FoldReport`'s clock dep).
- Error taxonomy closed: every `error.code` traces to exactly one exit class; `CONFIG_ERROR`'s 3
  raise-sites collapse to 1 code by design (not an inconsistency).
- Unit/component split matches the `component-tests` skill's boundary rule (R1-R9 = unit, adapter
  I/O failures = component) — verified in `contracts.md` §6's gate check (7 == 7).
- Ticket graph is acyclic and dependency-complete (ticket-23 depends on all prior; ticket-22 depends
  on every subordinate `ProcessValidate` calls).

## 3. Next step

Operator reviews this PLAN.md (+ linked artifacts) at **Gate #1**. On accept → `implementer` works
tickets 01-23 in dependency order (isolated worktree per `program-implementation`/harness
discipline). On reject/changes → back to `planner`/`plan-reviewer` per the fix-loop rules in the
harness `CLAUDE.md`.

<!-- DONE: PLAN slice-01-validate -->
