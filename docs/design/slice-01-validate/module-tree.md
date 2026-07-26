# slice-01-validate — module tree + head-pipe pseudocode

> Stage 5 (`program-design` Step 3) output for the slice's single Cockburn use case
> ([`use-case.md`](./use-case.md), UC1). Package: `internal/validate/`. C4 (C2+C3) →
> [`c4.md`](./c4.md); module contracts (Input/Deps/`io:`/antecedent/consequent) + unit-test
> formula + component-scenario set → [`contracts.md`](./contracts.md). Starting point:
> [`../../concept.md`](../../concept.md) §3–4 (already-reviewed C4 sketch + algorithm); this
> document formalizes it against the `program-design` hard rules (single `Request`, one data
> argument, valid-by-construction, `io:` tagging, file layout).

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
  pipe, no branching of its own (§4). `io: none`.
- **Egress adapter** `cli.ResolveExitCode(res: Result<Report, Error>) -> (exitCode int)` — maps
  the pipe's outcome to the exit-code grid (§5) + writes the report to stdout / logs to stderr.
  `io: none`.

## 3. The tree — subordinates

| # | Module | Kind | `io:` | Secret it hides (Parnas) |
|---|--------|------|-------|--------------------------|
| 1 | `cli.Parse` | ingress adapter | `none` | how the CLI invocation is spoken (cobra, argv) |
| 2 | `ProcessValidate` | head (ROP pipe) | `none` | the wiring order of an otherwise-designed pipe |
| 3 | `cli.ResolveExitCode` | egress adapter | `none` | how the outcome is serialized (exit code, stdout/stderr split) |
| 4 | `ConfigStore.Load` | I/O (fs read) | `none` | how `config.yaml` bytes are fetched from disk |
| 5 | `NewConfig` | domain constructor | `none` | `config.schema.json`'s validity rules |
| 6 | `ContractStore.Load` | I/O (fs read) | `none` | how `consumed-contract` bytes are fetched from disk |
| 7 | `NewConsumedContract` | domain constructor | `none` | `consumed-contract.schema.json`'s validity rules + consumer-name match |
| 8 | `BuildSpecLoader` | factory (logic) | `none` | which loader strategy applies (`spec_path` XOR `spec_url`) |
| 9 | `FileSpecLoader.Load` | I/O (fs read) | `none` | how a local spec file is fetched from disk |
| 10 | `HTTPSpecLoader.Load` | I/O (HTTP GET) | `http` | how a remote spec is fetched (bearer token from env, timeout) |
| 11 | `InlineServerRefs` | logic (pre-resolve) | `none` | the parser-library defect workaround (D2, `concept.md`) |
| 12 | `DeriveProviderChannels` | logic (projection) | `none` | the two AsyncAPI-3.0 defaults to expand (no-`messages`, `reply`) (D3) |
| 13 | `NewComparison` | uniting constructor | `none` | how `Config` + `ConsumedContract` + provider projection unite into one valid whole (incl. the channel-scope cross-check, FRD §4.1) |
| 14 | `ResolveChannelDirection` | logic (R1–R4) | `none` | channel/direction resolution rules |
| 15 | `CompareMessage` | logic (R5–R9) | `none` | message-level variance/type/contentType/correlationId rules |
| 16 | `CompareContracts` | logic (fold) | `none` | how per-channel/per-message violations accumulate into one `Outcome` |
| 17 | `FoldReport` | logic (DTO assembly) | `none` | canon-`1.1` report shape + where `generated_at` comes from |
| 18 | `BuildReportWriter` | factory (logic) | `none` | whether/where the report additionally lands on disk |
| 19 | `ReportWriter.Write` | I/O (fs write) | `none` | how report bytes are persisted |

19 modules; `io:` types used: **none** (18) + **http** (1, `HTTPSpecLoader.Load`). No `db`/`llm`/
`queue` module — this slice touches no database, LLM, or broker (concept.md §6, "Не делает").

### Notes on hard-rule compliance

- **One data argument.** Every node above takes exactly one data entity, with three documented,
  sanctioned exceptions:
  1. **`NewComparison(cfg, contract, pchans)`** — this *is* the "introduce a domain entity and a
     constructor node `NewT(...)`" the hard rule calls for: its entire job is uniting three
     already-valid parts into one `Comparison`. A uniting constructor's inputs are necessarily
     plural; that is the sanctioned form, not a violation.
  2. **`NewConsumedContract(raw, expectedConsumer)`** — `expectedConsumer` (`cfg.Consumer.Name`)
     is a scalar config value carried in `Dependencies:`, not a second data entity (allowed per
     `program-design` Step 5, "config value" row) — it does not need a domain type of its own.
  3. **`FoldReport(outcome)` + `Dependencies: clock.Clock`** — `generated_at` is obtained by
     `FoldReport` calling `deps.Clock.Now()` once, itself, as an injected dependency (allowed row:
     `clock.Clock` — deterministic time, not an integration), **not** as a second `Input`. This is
     the concrete mechanism behind D10 ("the core never reads system time"): the *only* clock read
     in the whole slice happens here, through the port, never via `time.Now()` in the core.
- **Invariant = subtype, not guard.** `NewConfig`, `NewConsumedContract` are subtype constructors
  (`Result<Config, Error>` / `Result<ConsumedContract, Error>`) — invalid input is never
  constructed, not merely flagged.
- **Valid by construction.** `Config`, `ConsumedContract`, `Comparison`, `Report` have unexported
  fields, built only through their `NewT` factory; no naked composite literal of these types
  appears outside `logic.go`.
- **I/O isolation.** All four external touchpoints (config file, consumed-contract file, provider
  spec file-or-HTTP, report file) are autonomous objects (`Store`/`Client`-shaped: `ConfigStore`,
  `ContractStore`, `FileSpecLoader`/`HTTPSpecLoader` behind one `SpecLoader` interface,
  `ReportWriter`); the head's `Deps` holds only these objects + `clock.Clock`, never a raw
  `*os.File`/`*http.Client`.
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
  its own pure, independently unit-tested logic module (row 11) rather than left inline, so the
  parser-defect workaround is swappable/removable in isolation once upstream fixes it (D2).
- **`CONFIG_ERROR` reachable from three places, one code.** `ConfigStore.Load` (file
  unreadable), `NewConfig` (schema breach), and `NewComparison` (channel scope not covered by the
  consumed-contract, FRD §4.1) can each raise `ErrConfigInvalid` → `CONFIG_ERROR`, exit 2, no
  report written. This is intentional (Extension 2a bundles all three under one use-case row,
  `use-case.md` Extensions) — the pipe never converts the error class, it only stops before
  `FoldReport`/`ReportWriter.Write` run.

## 4. Head-pipe pseudocode

```
ProcessValidate(inv: Invocation, deps: Deps) -> Result<Report, Error>:
  | ConfigStore.Load(inv.ConfigPath)                       -> RawConfig         # fs        [CONFIG_ERROR -> 2]
  | NewConfig(raw)                                          -> Config           # validate  [CONFIG_ERROR -> 2]
  | ContractStore.Load(cfg.Consumer.ConsumedContractPath)   -> RawContract      # fs        [FILE_NOT_FOUND -> 3]
  | NewConsumedContract(raw, cfg.Consumer.Name)             -> ConsumedContract # validate  [PARSE_ERROR -> 3]
  | BuildSpecLoader(cfg.Provider, cfg.Settings.Timeout)     -> SpecLoader       # late-bound factory (D … EMULATION S15)
  | SpecLoader.Load(cfg.Provider)                           -> ProviderSpec     # fs XOR http
  |     (internally: InlineServerRefs -> parser.FromYAML(...).Process())       [FILE_NOT_FOUND | PARSE_ERROR | HTTP_ERROR | TIMEOUT_ERROR -> 3]
  | DeriveProviderChannels(spec)                            -> ProviderChannels # projection: 2 defaults expanded (D3)
  | NewComparison(cfg, contract, pchans)                    -> Comparison       # unite      [CONFIG_ERROR -> 2] (channel-scope cross-check)
  | CompareContracts(comparison)                            -> Outcome          # CORE: R1..R9 fold, never short-circuits across channels
  | FoldReport(outcome)                                     -> Report           # canon 1.1; deps.Clock.Now() called exactly once (D10)
  | BuildReportWriter(cfg.Settings)                         -> ReportWriter     # factory: whether/where to persist
  | ReportWriter.Write(report)                              -> Report           # iff save_json_report; pass-through
  -> Ok(report)

then in main: code := cli.ResolveExitCode(result); report ALWAYS -> stdout; logs -> stderr; os.Exit(code)
```

A failing step short-circuits the remaining pipe; the error rises untransformed to
`cli.ResolveExitCode`, which is the **only** place an error class is mapped to an exit code
(`program-design` Step 3, "Errors — ROP short-circuit"). `CompareContracts` itself never returns
an error on the domain axis — `incompatible` is a legitimate value of `Outcome`, not an `Err`
(use-case.md, note after Extensions 6a–6i).

## 5. Exit-code / error-class grid (from `concept.md` §4, `report.schema.json` `x-exit-codes`)

| Exit | Class | Codes | Report written? |
|---|---|---|---|
| 0 | compatible | — | yes, `compatible: true`, `errors: []` |
| 1 | incompatible (verdict) | `CHANNEL_NOT_IN_PROVIDER`, `PROTOCOL_MISMATCH`, `DIRECTION_NOT_IN_PROVIDER`, `MESSAGE_NOT_IN_PROVIDER`, `MISSING_REQUIRED_SENT_FIELD`, `READS_FIELD_NOT_PROVIDED`, `TYPE_MISMATCH`, `CONTENT_TYPE_MISMATCH`, `CORRELATION_ID_MISMATCH` | yes |
| 2 | config | `CONFIG_ERROR` | **no** |
| 3 | io · parse | `FILE_NOT_FOUND`, `PARSE_ERROR`, `HTTP_ERROR`, `TIMEOUT_ERROR` | yes (`errors[]` carries the code) |

## 6. File layout (`internal/validate/`)

| File | Node(s) |
|---|---|
| `head.go` | `ProcessValidate` |
| `adapter.go` | `cli.Parse`, `cli.ResolveExitCode` |
| `domain.go` | `Invocation`, `Config`, `ConsumedContract`, `Comparison`, `Outcome`, `Report` + value types |
| `errors.go` | sentinel errors + `code → exit` table |
| `logic.go` | `NewConfig`, `NewConsumedContract`, `InlineServerRefs`, `DeriveProviderChannels`, `NewComparison`, `ResolveChannelDirection`, `CompareMessage`, `CompareContracts`, `FoldReport`, `BuildSpecLoader`, `BuildReportWriter` |
| `io_config.go` | `ConfigStore.Load` |
| `io_contract.go` | `ContractStore.Load` |
| `io_spec.go` | `SpecLoader` interface, `FileSpecLoader.Load`, `HTTPSpecLoader.Load` |
| `io_report.go` | `ReportWriter.Write` |
| `register.go` | `Deps{Clock clock.Clock, HTTPClient *http.Client}` + wiring to `cmd/pinout-asyncapi` |

Four I/O modules is the slice's natural complexity (config file, contract file, provider
spec file-or-HTTP, report file) — one external dependency and one mode (read or write) each
(`program-design` Step 6); not a reason to split the slice (one use case, one external input).

<!-- DONE: module-tree slice-01-validate -->
