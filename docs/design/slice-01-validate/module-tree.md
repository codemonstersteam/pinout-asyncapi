# slice-01-validate — module tree + head-pipe pseudocode

> Stage 5 (`program-design` Step 3) output for the slice's single Cockburn use case
> ([`use-case.md`](./use-case.md), UC1). Package: `internal/validate/`. C4 (C2+C3) →
> [`c4.md`](./c4.md); module contracts (Input/Deps/`io:`/antecedent/consequent) + unit-test
> formula + component-scenario set → [`contracts.md`](./contracts.md); load-bearing decisions →
> [`adr/`](./adr/). Starting point: [`../../concept.md`](../../concept.md) §3–4 (already-reviewed
> C4 sketch + algorithm); this document formalizes it against the `program-design` hard rules
> (single `Request`, one data argument, valid-by-construction, `io:` tagging, file layout).
>
> **Current as of change [`001-pipe-arity-coverage-gaps`](./changes/001-pipe-arity-coverage-gaps/)**
> (lane `patch`) — the pipe-arity reshape (19 → 21 nodes, bind-then-chain head) is folded in below;
> the change folder keeps the immutable per-change record ([`change-delta.md`](./changes/001-pipe-arity-coverage-gaps/change-delta.md)).

## 1. Slice = one module, one contract

1 external input (one-shot CLI invocation `pinout-asyncapi validate <config.yaml>`) → 1
`Request` (`Invocation{ConfigPath string}`) → 1 `Result<Report, Error>`. Antecedent: `ConfigPath`
is a non-empty argv string (the path itself may be unreadable — that's a pipe failure, not an
ingress-parse failure). Consequent: exactly one canon-`1.1` report on stdout (+ optionally to
file) and exactly one exit code 0/1/2/3 (use-case.md, Minimal guarantee).

## 2. Ingress → head

- **Ingress adapter** `cli.Parse(args []string) -> Invocation` — argv → the one flat `Request`.
  No flags beyond `--help`/`--version` (boilerplate, `cli-io`); the domain parameter (config
  path) is the sole positional arg. `io: none`.
- **Head** `ProcessValidate(inv: Invocation, deps: Deps) -> Result<Report, Error>` — linear ROP
  pipe, no branching of its own (§4). Signature and `Deps{Clock, HTTPClient}` are stable across
  the arity reshape — the head is callable identically before and after. `io: none`.
- **Egress adapter** `cli.ResolveExitCode(res: Result<Report, Error>) -> (exitCode int)` — maps
  the pipe's outcome to the exit-code grid (§5) + writes the report to stdout / logs to stderr.
  **Total over any `error`** — the `default:` arm is the out-of-taxonomy fallback → **3**
  (`contracts.md` §`cli.ResolveExitCode`). `io: none`.

## 3. The tree — subordinates

| # | Module | Kind | `io:` | Secret it hides (Parnas) |
|---|--------|------|-------|--------------------------|
| 1 | `cli.Parse` | ingress adapter | `none` | how the CLI invocation is spoken (cobra, argv) |
| 2 | `ProcessValidate` | head (ROP pipe) | `none` | the wiring order of an otherwise-designed pipe |
| 3 | `cli.ResolveExitCode` | egress adapter | `none` | how the outcome is serialized (exit code, stdout/stderr split), incl. the out-of-taxonomy fallback |
| 4 | `ConfigStore.Load` | I/O (fs read) | `none` | how `config.yaml` bytes are fetched from disk |
| 5 | `NewConfig` | domain constructor | `none` | `config.schema.json`'s validity rules |
| 6 | `ContractStore.Load` | I/O (fs read) | `none` | how `consumed-contract` bytes are fetched from disk |
| 7 | `BuildContractParser` | factory (logic) | `none` | which consumer identity the parse is bound to — the parse step never sees `Config` |
| 8 | `ContractParser.Parse` | domain constructor | `none` | `consumed-contract.schema.json`'s validity rules + consumer-name match |
| 9 | `BuildSpecLoader` | factory (logic) | `none` | which loader strategy applies (`spec_path` XOR `spec_url`) |
| 10 | `FileSpecLoader.Load` | I/O (fs read) | `none` | how a local spec file is fetched from disk |
| 11 | `HTTPSpecLoader.Load` | I/O (HTTP GET) | `http` | how a remote spec is fetched (bearer token from env, timeout) |
| 12 | `InlineServerRefs` | logic (pre-resolve) | `none` | the parser-library defect workaround (D2, `concept.md`) |
| 13 | `DeriveProviderChannels` | logic (projection) | `none` | the two AsyncAPI-3.0 defaults to expand (no-`messages`, `reply`) (D3) |
| 14 | `NewComparison` | uniting constructor | `none` | how the `ComparisonInput` join becomes one valid whole (incl. the channel-scope cross-check, FRD §4.1) |
| 15 | `ResolveChannelDirection` | logic (R1–R4) | `none` | channel/direction resolution rules |
| 16 | `CompareMessage` | logic (R5–R9) | `none` | message-level variance/type/contentType/correlationId rules |
| 17 | `CompareContracts` | logic (fold) | `none` | how per-channel/per-message violations accumulate into one `Outcome` |
| 18 | `BuildReporter` | factory (logic) | `none` | where `generated_at` comes from — the clock port, bound once, outside the pipe (D10) |
| 19 | `Reporter.Fold` | logic (DTO assembly) | `none` | canon-`1.1` report shape |
| 20 | `BuildReportWriter` | factory (logic) | `none` | whether/where the report additionally lands on disk |
| 21 | `ReportWriter.Write` | I/O (fs write) | `none` | how report bytes are persisted |

**21 modules**; `io:` types used: **none** (20) + **http** (1, `HTTPSpecLoader.Load`). No
`db`/`llm`/`queue` module — this slice touches no database, LLM, or broker (concept.md §6, "Не
делает").

### Factory and product-method are two nodes (ADR-001)

The tree contains four bind-then-chain pairs and splits every one of them, with zero
counterexamples:

- `BuildContractParser` (7) / `ContractParser.Parse` (8)
- `BuildSpecLoader` (9) / `FileSpecLoader.Load` (10) / `HTTPSpecLoader.Load` (11)
- `BuildReporter` (18) / `Reporter.Fold` (19)
- `BuildReportWriter` (20) / `ReportWriter.Write` (21)

The factory and the product-method hide **different** decisions, which is the whole test (Parnas):
`BuildSpecLoader` hides *which strategy*, `FileSpecLoader.Load` hides *how a file is read*;
`BuildReporter` hides *where time comes from*, `Reporter.Fold` hides *the canon-1.1 shape*.
Merging them would put two independently-changeable decisions behind one interface. Rationale →
[`adr/001-factory-and-product-method-are-two-nodes.md`](./adr/001-factory-and-product-method-are-two-nodes.md).

Each branchless logic factory carries its own unit-test row at N=1, exactly as `BuildSpecLoader`
(N=2) and `BuildReportWriter` (N=2) do; this is **not** a carve-out of the frozen "head, I/O
modules and adapters are not unit-tested" rule (no `adapter_test.go`, no head row, no I/O row).
Slice total: **51** unit tests (`contracts.md` §4).

### Node numbering history (change 001)

Artifacts written against the pre-`001` tree cite the old numbers; this map reads them.

| old # | old name | new # | new name |
|---|---|---|---|
| 1–6 | (unchanged) | 1–6 | (unchanged) |
| 7 | `NewConsumedContract` | **7 + 8** | `BuildContractParser` + `ContractParser.Parse` |
| 8 | `BuildSpecLoader` | 9 | `BuildSpecLoader` |
| 9 | `FileSpecLoader.Load` | 10 | `FileSpecLoader.Load` |
| 10 | `HTTPSpecLoader.Load` | 11 | `HTTPSpecLoader.Load` |
| 11 | `InlineServerRefs` | 12 | `InlineServerRefs` |
| 12 | `DeriveProviderChannels` | 13 | `DeriveProviderChannels` |
| 13 | `NewComparison` | 14 | `NewComparison` (arg → `ComparisonInput`) |
| 14 | `ResolveChannelDirection` | 15 | `ResolveChannelDirection` |
| 15 | `CompareMessage` | 16 | `CompareMessage` |
| 16 | `CompareContracts` | 17 | `CompareContracts` |
| 17 | `FoldReport` | **18 + 19** | `BuildReporter` + `Reporter.Fold` |
| 18 | `BuildReportWriter` | 20 | `BuildReportWriter` |
| 19 | `ReportWriter.Write` | 21 | `ReportWriter.Write` |

### Notes on hard-rule compliance

- **One data argument — no exceptions.** Every pipe node above takes exactly **one** data entity;
  the "sanctioned exceptions by arity" list is **empty**:
  1. `NewComparison` — the three-stream union is materialized as the named join type
     `ComparisonInput{Config, Contract, ProviderChannels}` (`contracts.md` §1). `NewComparison`
     takes **one** argument; the join is visible in the type system instead of in an arity.
  2. The consumed-contract parse — `expectedConsumer` lives at construction
     (`BuildContractParser(consumerName)`), and `ContractParser.Parse(RawContract)` takes one data
     entity.
  3. The report fold — the clock port lives at construction (`BuildReporter(clock)`), and
     `Reporter.Fold(Outcome)` takes one data entity.
  **Scope of the rule:** it governs *pipe steps*, not factory construction. A factory is not a pipe
  step — it is the bind that *removes* arity from the pipe, and its construction parameters
  (`BuildSpecLoader(provider, timeout, client)`) are the sanctioned form named in the rule itself.
  That is precisely why the list can be empty rather than merely shorter.
- **`ComparisonInput` is a join, not a shared `Context`/`State`.** It is assembled once, at exactly
  the one point where three flows genuinely unite, is consumed by exactly one node
  (`NewComparison`), and is never threaded further down the pipe — `CompareContracts` receives the
  validated `Comparison`, not the input. It carries three fields and no ports, no config bag, no
  clock. A carrier that grew a fourth unrelated field, or that survived past `NewComparison`, would
  be the counter-rule violation; this one does not.
- **Invariant = subtype, not guard.** `NewConfig`, `ContractParser.Parse`, `NewComparison` are
  subtype constructors (`Result<Config, Error>` / `Result<ConsumedContract, Error>` /
  `Result<Comparison, Error>`) — invalid input is never constructed, not merely flagged. `Parse` is
  the only way a `ConsumedContract` comes to exist.
- **Valid by construction.** `Config`, `ConsumedContract`, `Comparison`, `Report` have unexported
  fields, built only through their constructor; no naked composite literal of these types appears
  outside `logic.go`. `ComparisonInput` is deliberately **not** in this set — it is a plain
  transport DTO with exported fields whose validity is established by `NewComparison`, the node it
  feeds. `Reporter` and `ContractParser` are collaborators, not domain entities: unexported fields,
  built only by their factory.
- **I/O isolation.** All four external touchpoints (config file, consumed-contract file, provider
  spec file-or-HTTP, report file) are autonomous objects (`Store`/`Client`-shaped: `ConfigStore`,
  `ContractStore`, `FileSpecLoader`/`HTTPSpecLoader` behind one `SpecLoader` interface,
  `ReportWriter`); the head's `Deps` holds only these objects + `clock.Clock`, never a raw
  `*os.File`/`*http.Client`. `Reporter` and `ContractParser` are **not** I/O objects — they perform
  no I/O; they are pure collaborators that happen to be bound before use.
- **One integration per I/O module.** `SpecLoader` is modeled as **two** objects behind one
  interface (Strategy), not one object branching on `spec_path`/`spec_url` internally — `NewConfig`
  already guarantees exactly one of the two is set (`oneOf` in the schema), so `BuildSpecLoader`
  picks the implementation once, and each implementation then has exactly one external dependency
  and one `io:` tag (`none` for file, `http` for HTTP), per Step 6's "one dependency, one mode" rule.
- **Pipe-embedded logic in an I/O module (documented exception).** `FileSpecLoader.Load` /
  `HTTPSpecLoader.Load` call `InlineServerRefs` (a pure pre-processing step, D2) and the stock
  parser's `FromYAML(...).Process()` (decode into the library's typed model) before returning.
  This is the minimum necessary "fetch bytes → produce the typed external representation" a
  loader must do; it is not business-logic branching. `InlineServerRefs` itself is factored out as
  its own pure, independently unit-tested logic module (row 12) rather than left inline, so the
  parser-defect workaround is swappable/removable in isolation once upstream fixes it (D2).
- **`CONFIG_ERROR` reachable from three places, one code.** `ConfigStore.Load` (file
  unreadable), `NewConfig` (schema breach), and `NewComparison` (channel scope not covered by the
  consumed-contract, FRD §4.1) can each raise `ErrConfigInvalid` → `CONFIG_ERROR`, exit 2, no
  report written. This is intentional (Extension 2a bundles all three under one use-case row,
  `use-case.md` Extensions) — the pipe never converts the error class, it only stops before
  `Reporter.Fold`/`ReportWriter.Write` run.
- **One clock read.** D10: the single `Now()` call lives *inside* `Reporter`, reached only through
  the port bound at `BuildReporter`. `grep -rn 'time.Now()' internal/validate/` must match only
  `register.go`, and a counting fake must observe exactly **1** call per `Fold`.

## 4. Head-pipe pseudocode — bind-then-chain

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

**Why bind-then-chain is observationally identical to a flat pipe.** All four binds are total —
`BuildContractParser`, `BuildSpecLoader`, `BuildReporter`, `BuildReportWriter` have no failure
branch and perform no I/O — so hoisting them ahead of the chain cannot change *which* error
surfaces first, nor the order in which the four I/O touchpoints happen. The observable sequence of
failures, the exit-code grid and the report bytes are pinned by two byte anchors: the baseline
stdout fixture (`fixtures/validate/good/report.baseline.json`) and the `ProcessValidate` golden
(`testdata/report.golden.json`).

A failing step short-circuits the remaining pipe; the error rises untransformed to
`cli.ResolveExitCode`, which is the **only** place an error class is mapped to an exit code
(`program-design` Step 3, "Errors — ROP short-circuit"). `CompareContracts` itself never returns
an error on the domain axis — `incompatible` is a legitimate value of `Outcome`, not an `Err`
(use-case.md, note after Extensions 6a–6i); it becomes exit **1** at `cli.ResolveExitCode`, the
grid row that component scenario 9 proves through the binary (`contracts.md` §5/§6).

## 5. Exit-code / error-class grid (from `concept.md` §4, `report.schema.json` `x-exit-codes`)

| Exit | Class | Codes | Report written? |
|---|---|---|---|
| 0 | compatible | — | yes, `compatible: true`, `errors: []` |
| 1 | incompatible (verdict) | `CHANNEL_NOT_IN_PROVIDER`, `PROTOCOL_MISMATCH`, `DIRECTION_NOT_IN_PROVIDER`, `MESSAGE_NOT_IN_PROVIDER`, `MISSING_REQUIRED_SENT_FIELD`, `READS_FIELD_NOT_PROVIDED`, `TYPE_MISMATCH`, `CONTENT_TYPE_MISMATCH`, `CORRELATION_ID_MISMATCH` | yes |
| 2 | config | `CONFIG_ERROR` | **no** |
| 3 | io · parse | `FILE_NOT_FOUND`, `PARSE_ERROR`, `HTTP_ERROR`, `TIMEOUT_ERROR` | yes (`errors[]` carries the code) |

Coverage of the grid *through the binary* → `contracts.md` §5 ("C2 grid matrix"): 7/7.

An error outside this closed taxonomy (concretely: a `ReportWriter.Write` disk failure) is **not** a
fourth class — `cli.ResolveExitCode` is total and its `default:` arm maps it to **3**
(`contracts.md` §`cli.ResolveExitCode`). That path is pinned by a `cmd/app/main_test.go` case,
**not** by a component scenario.

## 6. File layout (`internal/validate/`)

| File | Node(s) |
|---|---|
| `head.go` | `ProcessValidate` |
| `adapter.go` | `cli.Parse`, `cli.ResolveExitCode` |
| `domain.go` | `Invocation`, `Config`, `ConsumedContract`, `Comparison`, `ComparisonInput`, `Outcome`, `Report` + value types |
| `errors.go` | sentinel errors + `code → exit` table |
| `logic.go` | `NewConfig`, `BuildContractParser`, `ContractParser.Parse`, `InlineServerRefs`, `DeriveProviderChannels`, `NewComparison`, `ResolveChannelDirection`, `CompareMessage`, `CompareContracts`, `BuildReporter`, `Reporter.Fold`, `BuildSpecLoader`, `BuildReportWriter` (+ the `ContractParser` / `Reporter` collaborator structs, next to their factories) |
| `io_config.go` | `ConfigStore.Load` |
| `io_contract.go` | `ContractStore.Load` |
| `io_spec.go` | `SpecLoader` interface, `FileSpecLoader.Load`, `HTTPSpecLoader.Load` |
| `io_report.go` | `ReportWriter.Write` |
| `register.go` | `Deps{Clock clock.Clock, HTTPClient *http.Client}` + wiring to `cmd/pinout-asyncapi` |

Every path roots in `internal/validate/`; no layer-keyed root is introduced.

Four I/O modules is the slice's natural complexity (config file, contract file, provider
spec file-or-HTTP, report file) — one external dependency and one mode (read or write) each
(`program-design` Step 6); not a reason to split the slice (one use case, one external input).

<!-- DONE: module-tree slice-01-validate -->
<!-- DONE: moduledesigner slice-01-validate -->
