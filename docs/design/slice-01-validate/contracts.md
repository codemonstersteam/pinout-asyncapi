# slice-01-validate — module contracts, message catalog, tests, component scenarios

> Companion to [`module-tree.md`](./module-tree.md) (Steps 4/5/6/8 of `program-design`); load-bearing
> decisions → [`adr/`](./adr/). Frozen contracts referenced throughout:
> [`config.schema.json`](../../../api-specification/config.schema.json),
> [`consumed-contract.schema.json`](../../../api-specification/consumed-contract.schema.json),
> [`report.schema.json`](../../../api-specification/report.schema.json).
>
> **Current as of change [`001-pipe-arity-coverage-gaps`](./changes/001-pipe-arity-coverage-gaps/)**
> (lane `patch`, frozen contracts untouched) — the reshaped node cards, the 51-unit total and the
> 9-scenario component set are folded in below; the change folder keeps the per-change record.

## 1. Message catalog (Step 4)

**Type ownership (data-deps).** Every type below is owned by **exactly one** module — the module that
constructs it. A module whose signature *consumes* a type depends on that type's owner; this column is
the source of the ticketer's `blocked_by` edges, alongside the call nesting in `module-tree.md` §4.

| Type | Kind | **Owner (constructs it)** | Notes |
|---|---|---|---|
| `Invocation` | `Request` | `cli.Parse` | `{ConfigPath string}` — unvalidated, from `cli.Parse`. |
| `RawConfig` | raw | `ConfigStore.Load` | decoded YAML/JSON bytes of `config.yaml`, pre-validation. |
| `Config` | `Entity` | `NewConfig` | valid-by-construction; unexported fields; built only via `NewConfig`. |
| `RawContract` | raw | `ContractStore.Load` | decoded bytes of the `consumed-contract` artifact, pre-validation. |
| `ContractParser` | collaborator | `BuildContractParser` | Unexported field: the expected consumer name (an already-`NewConfig`-validated config scalar). Performs no I/O. Exists so `Parse` takes one data entity. |
| `ConsumedContract` | `Entity` | `ContractParser.Parse` | valid-by-construction; built only via `Parse`. |
| `SpecLoader` | collaborator (interface) | `BuildSpecLoader` | `FileSpecLoader` XOR `HTTPSpecLoader`, chosen once at bind time. |
| `ProviderSpec` | `Entity` (external) | `FileSpecLoader.Load` / `HTTPSpecLoader.Load` | the `lerenn/asyncapi-codegen` typed AsyncAPI 3.0 model, post `InlineServerRefs` + `Process()`. |
| `ProviderChannels` | DTO | `DeriveProviderChannels` | `map[address]{protocol, send: []MessageRef, receive: []MessageRef}`. |
| `ComparisonInput` | DTO (named join) | assembled in the head; declared with `NewComparison` | `{Config Config, Contract ConsumedContract, ProviderChannels ProviderChannels}` — exported fields, no validity claim of its own. The one point in the slice where three flows genuinely unite, materialized as a type so `NewComparison` takes **one** argument (`module-tree.md` §3, "One data argument"). **Not** a shared `Context`/`State`: consumed by exactly one node and never threaded further; carries no ports, no clock, no config bag. |
| `Comparison` | `Entity` | `NewComparison` | valid-by-construction; the validated union of the `ComparisonInput`. |
| `Outcome` | DTO | `CompareContracts` | `{Violations []Violation, UncoveredChannels []string, ConsumerName string, Provenance Provenance}` — carries forward everything `Reporter.Fold` needs besides the clock. |
| `Reporter` | collaborator | `BuildReporter` | Unexported field: the `clock.Clock` port. Performs no I/O. Exists so `Fold` takes one data entity and so the slice's only clock read is enclosed (D10). |
| `Report` | DTO | `Reporter.Fold` | canon-`1.1` shape, `report.schema.json`. |
| `ReportWriter` | collaborator | `BuildReportWriter` | no-op internally when `save_json_report == false`. |
| `Error` | `Error` | `errors.go` (sentinels) | see §2. |

**Signature rule (Go idiom):** every `Result<T, Error>` below is a Go `(T, error)` return pair.

**Ownership edges (for the ticketer):** `ContractParser.Parse` depends on `BuildContractParser`;
`Reporter.Fold` depends on `BuildReporter`; `NewComparison` depends on the owners of all three
`ComparisonInput` fields (`NewConfig`, `ContractParser.Parse`, `DeriveProviderChannels`);
`ProcessValidate` depends on all four factories.

## 2. Error model

| `error.code` | Raised by | Exit | In `errors[]`? |
|---|---|---|---|
| `CONFIG_ERROR` | `ConfigStore.Load` (unreadable), `NewConfig` (schema breach), `NewComparison` (channel scope not in consumed-contract) | 2 | **no** — detected before `Reporter.Fold` runs |
| `FILE_NOT_FOUND` | `ContractStore.Load`; `FileSpecLoader.Load` | 3 | yes |
| `PARSE_ERROR` | `ContractParser.Parse`; `FileSpecLoader.Load`/`HTTPSpecLoader.Load` (body doesn't parse as AsyncAPI 3.0) | 3 | yes |
| `HTTP_ERROR` | `HTTPSpecLoader.Load` (unreachable / non-2xx) | 3 | yes |
| `TIMEOUT_ERROR` | `HTTPSpecLoader.Load` (exceeds `settings.timeout`) | 3 | yes |
| `CHANNEL_NOT_IN_PROVIDER` (R1) | `ResolveChannelDirection` | 1 | yes |
| `PROTOCOL_MISMATCH` (R2) | `ResolveChannelDirection` | 1 | yes |
| `DIRECTION_NOT_IN_PROVIDER` (R3) | `ResolveChannelDirection` | 1 | yes |
| `MESSAGE_NOT_IN_PROVIDER` (R4) | `ResolveChannelDirection` | 1 | yes |
| `MISSING_REQUIRED_SENT_FIELD` (R5) | `CompareMessage` | 1 | yes |
| `READS_FIELD_NOT_PROVIDED` (R6) | `CompareMessage` | 1 | yes |
| `TYPE_MISMATCH` (R7) | `CompareMessage` | 1 | yes |
| `CONTENT_TYPE_MISMATCH` (R8) | `CompareMessage` | 1 | yes |
| `CORRELATION_ID_MISMATCH` (R9) | `CompareMessage` | 1 | yes |

Visibility rule honored: `CONFIG_ERROR` is visible as a non-zero exit even though absent from
`errors[]`; nothing "couldn't check" is ever masked as `compatible: true`.
`errors[].subject`/`.location` — see D11 (`concept.md`): `location = subject + field path`,
computed once in `CompareMessage`/`ResolveChannelDirection` (verdict codes) or in the failing I/O
module (io/parse codes, subject = the artifact path/URL); `Reporter.Fold` never recomputes it.

**Out of the taxonomy (documented, not added):** a `ReportWriter.Write` disk/marshal failure wraps
**no** sentinel and has **no** `error.code` in the frozen enum. It is not a 14th row here — it is
the concrete reachable instance of `cli.ResolveExitCode`'s total-function fallback → exit **3**
(see the two cards below).

## 3. Module contracts

Only behavioral modules get a card; `domain.go`/`errors.go` (types/sentinels) and `register.go`
(wiring) do not (`program-design` Step 5).

### `cli.Parse`

- **Signature:** `Parse(args []string) -> Result<Invocation, Error>`
- **Input (data):** `args []string` (argv)
- **Dependencies:** —
- **io:** `none`
- **What it does:** parses the one positional arg into the flat `Request`.
- **Antecedent:** exactly one non-flag argument present (else usage error, not a domain `Error` — cobra's own `--help`/usage path, `cobra.ExactArgs(1)`).
- **Consequent:** success → `Invocation{ConfigPath}`. No domain failure branch (parsing only).
- **Note (defense-in-depth):** the "argc ≠ 1 never reaches the pipe" invariant is held by **two** independent guards — `cobra.ExactArgs(1)` and `Parse`'s own guard on `args[0]`. Neither is removed. It is proven through the binary (`cmd/app/main_test.go`, 0-arg and 2-arg cases), **not** by a unit test of the adapter (no `adapter_test.go`).
- **Note (scope of the "adapters are not unit-tested" rule):** the rule forbids unit-testing the **ingress adapter**, i.e. the package-level `cli.Parse` / `cli.ResolveExitCode` (in Go: `validate.Parse(` / `validate.ResolveExitCode(` of `internal/validate/adapter.go`, called **unqualified**). It does **not** reach `ContractParser.Parse(RawContract)` — the domain method (§`ContractParser.Parse`, ADR-001), which is core logic and **is** directly unit-tested by §4. Mechanical form: `grep -nE '(^|[^.[:alnum:]_])(Parse|ResolveExitCode)\(' internal/validate/*_test.go` must be empty; a receiver-qualified `.Parse(` is a different symbol and is deliberately excluded.

### `ProcessValidate`

- **Signature:** `ProcessValidate(inv: Invocation, deps: Deps) -> Result<Report, Error>`
- **Input (data):** `Invocation`
- **Dependencies:** `Deps{Clock clock.Clock, HTTPClient *http.Client}`
- **io:** `none`
- **What it does:** orchestrates the pipe as **bind-then-chain** (`module-tree.md` §4): four total factories are bound first (`BuildContractParser`, `BuildSpecLoader`, `BuildReporter`, `BuildReportWriter`), then a chain in which **every step takes exactly one data entity**. No logic of its own; no branching of its own.
- **Antecedent:** a well-formed `Invocation`.
- **Consequent:** success → `Report`; failure → the first short-circuiting step's `Error`, untransformed. The binds are total and perform no I/O, so the set of reachable errors, their order, the exit-code grid and the report bytes are what a flat pipe would produce.
- **Not unit-tested** (head, Step 8.1). The `internal/validate` determinism test that drives it (`testdata/report.golden.json`) is a **D10 determinism regression anchor, not a head unit test** — the same documented, non-formula status `internal/validate/report_json_shape_test.go` already carries. It adds **no** row to §4.

### `cli.ResolveExitCode`

- **Signature:** `ResolveExitCode(res: Result<Report, Error>) -> int` (Go: `ResolveExitCode(report Report, err error) int`)
- **Input (data):** `Result<Report, Error>` — the pipe outcome, one data entity
- **Dependencies:** —
- **io:** `none`
- **What it does:** maps the outcome to the exit-code grid (`module-tree.md` §5); prints the report to stdout on any branch that has one, logs to stderr.
- **Antecedent:** **none — the function is total over ANY `error`**, not merely over the closed sentinel taxonomy. The `default:` arm is the deliberate out-of-taxonomy fallback and it returns **3**.
  - **Concrete reachable instance:** a `ReportWriter.Write` disk/marshal failure (`io_report.go`) wraps no sentinel; it lands on `default:` → exit **3**. This is pinned by one case in `cmd/app/main_test.go`, **not** by a component scenario — a report-write failure is not one of the 7 adapter branches and a 10th scenario would break §6's gate.
  - **Deliberately unasserted (open debt):** on that path the synthesized stdout body carries `errors[0].code == ""` and no `subject`, which is **schema-invalid** against `report.schema.json`. Pinning or fixing it is a behaviour change requiring a frozen-contract decision; recorded as a follow-up debt item.
- **Consequent:** exactly one of `{0,1,2,3}`, for every possible input.
- **Not unit-tested** (adapter, Step 8.1 — no `adapter_test.go`, no §4 row). The full 7-row grid is proven **through the binary** instead; the matrix is in §5.

### `ConfigStore.Load`

- **Signature:** `(s ConfigStore) Load(path string) -> Result<RawConfig, Error>`
- **Input (data):** `path string`
- **Dependencies:** —
- **io:** `none` (filesystem pipe)
- **What it does:** reads `config.yaml` bytes and decodes YAML/JSON into `RawConfig`. Pure pipe — no validation.
- **Antecedent:** —
- **Consequent:** success → `RawConfig`; failure (missing/unreadable/undecodable) → `ErrConfigInvalid` (`CONFIG_ERROR`).

### `NewConfig`

- **Signature:** `NewConfig(raw: RawConfig) -> Result<Config, Error>`
- **Input (data):** `RawConfig`
- **Dependencies:** —
- **io:** `none`
- **What it does:** validates against `config.schema.json`'s rules; builds the valid-by-construction `Config`.
- **Antecedent:** `consumer.name` non-empty; `consumer.consumed_contract_path` non-empty; `consumer.channels` non-empty, unique, no empty elements; `provider.name` non-empty; exactly one of `provider.spec_path`/`provider.spec_url`; `settings.log_level` in enum; `settings.timeout > 0`; every other `settings` field within its declared type/range.
- **Consequent:** success → `Config`; failure (any antecedent clause) → `ErrConfigInvalid` (`CONFIG_ERROR`).

### `ContractStore.Load`

- **Signature:** `(s ContractStore) Load(path string) -> Result<RawContract, Error>`
- **Input (data):** `path string`
- **Dependencies:** —
- **io:** `none` (filesystem pipe)
- **What it does:** reads the `consumed-contract` artifact bytes and decodes into `RawContract`.
- **Antecedent:** —
- **Consequent:** success → `RawContract`; failure (missing/unreadable) → `ErrFileNotFound` (`FILE_NOT_FOUND`).

### `BuildContractParser`

- **Signature:** `BuildContractParser(consumerName string) -> ContractParser`
- **Input (data):** `consumerName string` (= `cfg.Consumer.Name`, an already-`NewConfig`-validated config scalar)
- **Dependencies:** —
- **io:** `none` (factory; performs no I/O)
- **What it does:** binds the expected consumer identity once, out of the pipe, so the parse step takes exactly one data entity and never sees `Config`.
- **Antecedent:** `consumerName` non-empty (guaranteed by `NewConfig`'s antecedent).
- **Consequent:** a `ContractParser` carrying that name. **No failure branch** (construction only) — this is why binding it ahead of the chain is behaviour-preserving.

### `ContractParser.Parse`

- **Signature:** `(p ContractParser) Parse(raw: RawContract) -> Result<ConsumedContract, Error>`
- **Input (data):** `RawContract` — **one** data entity (the expected-consumer argument lives at construction)
- **Dependencies:** — (the expected consumer name is captured in the receiver, not passed)
- **io:** `none`
- **What it does:** validates against `consumed-contract.schema.json`; builds the valid-by-construction `ConsumedContract`.
- **Antecedent:** decodes as valid YAML/JSON; `schema_version == "1.0"`; `consumer == p.expectedConsumer`; `provenance.captured_hash` matches `^sha256:[0-9a-f]{64}$`; `channels` non-empty; each channel has non-empty `address`/`protocol` and at least one of `sends`/`receives`.
- **Consequent:** success → `ConsumedContract`; failure (any antecedent clause) → `ErrContractInvalid` → `PARSE_ERROR` → exit 3.
- **Subtype, not guard:** the only way a `ConsumedContract` comes to exist.

### `BuildSpecLoader`

- **Signature:** `BuildSpecLoader(provider: ProviderConfig) -> SpecLoader`
- **Input (data):** `ProviderConfig` (the `Config.Provider` sub-value: name + exactly one of `spec_path`/`spec_url`)
- **Dependencies:** `timeout time.Duration` (= `cfg.Settings.Timeout`, config value); `HTTPClient *http.Client` (only used if the HTTP branch is selected — encapsulated into the returned `HTTPSpecLoader`, never exposed to the head)
- **io:** `none` (factory; performs no I/O itself)
- **What it does:** selects and constructs exactly one loader implementation, late-bound so `timeout` always comes from the already-validated `Config` (never a constant — EMULATION.md S15 guards this).
- **Antecedent:** exactly one of `spec_path`/`spec_url` set (guaranteed by `NewConfig`'s antecedent).
- **Consequent:** a `SpecLoader` (interface) — `FileSpecLoader` when `spec_path` set, `HTTPSpecLoader` when `spec_url` set. No failure branch (construction only).

### `FileSpecLoader.Load`

- **Signature:** `(l FileSpecLoader) Load() -> Result<ProviderSpec, Error>`
- **Input (data):** void (path captured at construction)
- **Dependencies:** —
- **io:** `none` (filesystem pipe)
- **What it does:** reads the spec file, applies `InlineServerRefs`, hands bytes to `parser.FromYAML(...).Process()`.
- **Antecedent:** —
- **Consequent:** success → `ProviderSpec`; failure: file missing/unreadable → `ErrFileNotFound` (`FILE_NOT_FOUND`); doesn't parse as AsyncAPI 3.0 → `ErrParseError` (`PARSE_ERROR`).

### `HTTPSpecLoader.Load`

- **Signature:** `(l HTTPSpecLoader) Load() -> Result<ProviderSpec, Error>`
- **Input (data):** void (URL + timeout captured at construction)
- **Dependencies:** `*http.Client` (encapsulated); `PINOUT_PROVIDER_TOKEN` read from env internally (never logged, never in the domain types)
- **io:** `http`
- **What it does:** `GET spec_url` with `Authorization: Bearer $PINOUT_PROVIDER_TOKEN` when set, bounded by `timeout`; applies `InlineServerRefs`; hands the body to `parser.FromYAML(...).Process()`.
- **Antecedent:** —
- **Consequent:** success → `ProviderSpec`; failure: unreachable/non-2xx → `ErrHTTPError` (`HTTP_ERROR`); exceeds `timeout` → `ErrTimeoutError` (`TIMEOUT_ERROR`); body doesn't parse → `ErrParseError` (`PARSE_ERROR`).

### `InlineServerRefs`

- **Signature:** `InlineServerRefs(doc: YAMLTree) -> YAMLTree`
- **Input (data):** decoded YAML tree of the provider spec
- **Dependencies:** —
- **io:** `none`
- **What it does:** substitutes `#/servers/*` `$ref`s with the referenced server body, in the tree, before the stock parser runs (D2 workaround).
- **Antecedent:** a decoded YAML tree (any shape; a malformed document surfaces later at `Process()`, not here).
- **Consequent:** the same tree with every `#/servers/*` ref inlined; a no-op when there are none.

### `DeriveProviderChannels`

- **Signature:** `DeriveProviderChannels(spec: ProviderSpec) -> ProviderChannels`
- **Input (data):** `ProviderSpec`
- **Dependencies:** —
- **io:** `none`
- **What it does:** projects the parsed spec into `address → {protocol, send: [msg], receive: [msg]}`, expanding the "no `messages`" default (= all channel messages) and the `reply` default (D3).
- **Antecedent:** a `ProviderSpec` produced by a successful `Process()` (typed, refs resolved).
- **Consequent:** `ProviderChannels`, total over any valid `ProviderSpec` — no failure branch.

### `NewComparison`

- **Signature:** `NewComparison(in: ComparisonInput) -> Result<Comparison, Error>`
- **Input (data):** `ComparisonInput{Config, Contract, ProviderChannels}` — **one** data entity. This *is* the "materialize a real join as a named input type with its own constructor node" the hard rule prescribes; the arity is not removed, it is **named**. The node stays a single constructor.
- **Dependencies:** —
- **io:** `none`
- **What it does:** unites the three already-valid parts into one valid `Comparison`; performs the one cross-artifact check that can only run here (FRD §4.1: every address in `cfg.Consumer.Channels` must appear in `contract.Channels`).
- **Antecedent:** every address in `in.Config.Consumer.Channels` is present among `in.Contract.Channels[].Address`.
- **Consequent:** success → `Comparison`; failure → `ErrConfigInvalid` (`CONFIG_ERROR`) → exit 2 — still before any report is written.
- **Fixture trap this card exists to prevent:** the `incompatible/` fixture's extra channel address MUST be present in the consumed-contract too. An address in the config but absent from the contract trips **this** antecedent → exit **2**, not the exit **1** verdict scenario 9 is meant to prove.

### `ResolveChannelDirection`

- **Signature:** `ResolveChannelDirection(ch: ChannelComparison) -> (resolved ResolvedDirection, violations []Violation)`
- **Input (data):** `ChannelComparison` (one consumer channel + its provider-projected counterpart, per direction)
- **Dependencies:** —
- **io:** `none`
- **What it does:** R1–R4 — channel presence, protocol match, counterpart-direction presence, message-key presence.
- **Antecedent:** a `ChannelComparison` for one address (channel resolution is hierarchical-stop: no channel ⇒ no direction check; no direction ⇒ no message check, EMULATION.md §3).
- **Consequent:** the resolved message set to hand to `CompareMessage`, plus zero or more of `{CHANNEL_NOT_IN_PROVIDER, PROTOCOL_MISMATCH, DIRECTION_NOT_IN_PROVIDER, MESSAGE_NOT_IN_PROVIDER}`.

### `CompareMessage`

- **Signature:** `CompareMessage(m: MessageComparison) -> []Violation`
- **Input (data):** `MessageComparison` (one message pair, matched by `channel.messages` map key — D8 — with its direction's variance rule)
- **Dependencies:** —
- **io:** `none`
- **What it does:** R5–R9 over `payload` and `headers` alike — contravariant/covariant field-set inclusion, recursive type/format match, `contentType` (iff declared), `correlationId.location` (iff declared, D9).
- **Antecedent:** a resolved `MessageComparison` (message key matched on both sides).
- **Consequent:** zero or more of `{MISSING_REQUIRED_SENT_FIELD, READS_FIELD_NOT_PROVIDED, TYPE_MISMATCH, CONTENT_TYPE_MISMATCH, CORRELATION_ID_MISMATCH}`.

### `CompareContracts`

- **Signature:** `CompareContracts(c: Comparison) -> Outcome`
- **Input (data):** `Comparison`
- **Dependencies:** —
- **io:** `none`
- **What it does:** folds `ResolveChannelDirection` + `CompareMessage` over every channel (sorted order, determinism) in `consumer.channels`, accumulating violations without short-circuiting across channels; also computes `uncovered_channels[]` (provider channels outside `consumer.channels`) and carries forward `ConsumerName`/`Provenance` for `Reporter.Fold`.
- **Antecedent:** a valid `Comparison`.
- **Consequent:** `Outcome{Violations, UncoveredChannels, ConsumerName, Provenance}` — never an `Error`; `incompatible` is a **value**, not a failure (use-case.md, note after 6a–6i). It becomes exit **1** only at `cli.ResolveExitCode`.

### `BuildReporter`

- **Signature:** `BuildReporter(c: clock.Clock) -> Reporter`
- **Input (data):** the `clock.Clock` port — bound at construction, **not** passed through the pipe
- **Dependencies:** —
- **io:** `none` (factory; performs no I/O)
- **What it does:** encloses the time source so that `Fold` takes exactly one data entity and the slice's only clock read is unreachable from anywhere else.
- **Antecedent:** a non-nil `clock.Clock` (supplied by the head from `Deps`).
- **Consequent:** a `Reporter` bound to that clock. **No failure branch** (construction only).
- **D10 invariant:** `grep -rn 'time.Now()' internal/validate/` must match only `register.go`; a counting fake must observe exactly **1** `Now()` call across one `Fold`.

### `Reporter.Fold`

- **Signature:** `(r Reporter) Fold(outcome: Outcome) -> Report`
- **Input (data):** `Outcome` — **one** data entity
- **Dependencies:** — (the clock is captured in the receiver; it calls `.Now()` exactly once, the whole slice's only clock read, D10)
- **io:** `none`
- **What it does:** assembles the canon-`1.1` `Report`: `compatible ⇔ errors == []`, `errors[]` (from `Violations`), `provenance` (echoed), `uncovered_channels[]`, `generated_at` (from the bound clock), `validator`/`interaction` build-time constants.
- **Antecedent:** a valid `Outcome`.
- **Consequent:** `Report`, total — no failure branch. Byte-pinned by the stdout baseline + the `ProcessValidate` golden.

### `BuildReportWriter`

- **Signature:** `BuildReportWriter(settings: Settings) -> ReportWriter`
- **Input (data):** `Settings` (`save_json_report`, `json_report_file`)
- **Dependencies:** —
- **io:** `none`
- **What it does:** constructs the writer, encapsulating whether/where to persist (Store-style: the head never branches on `save_json_report` itself).
- **Antecedent:** —
- **Consequent:** a `ReportWriter` (no-op internally when `save_json_report == false`). No failure branch.

### `ReportWriter.Write`

- **Signature:** `(w ReportWriter) Write(report: Report) -> Result<Report, Error>`
- **Input (data):** `Report`
- **Dependencies:** —
- **io:** `none` (filesystem pipe)
- **What it does:** writes the report bytes to the configured path (or no-op, per construction); always returns the report unchanged for the pipe to continue.
- **Antecedent:** —
- **Consequent:** success → `Report` (pass-through). A disk-write/marshal failure here wraps **no** sentinel and has **no code in the frozen error taxonomy** (`report.schema.json` `x-exit-codes` has none for it) — out of scope for this design (frozen contracts are not modified by this stage). It is nevertheless a *reachable* path, and it is exactly the concrete instance that exercises `cli.ResolveExitCode`'s total-function fallback → exit **3** (see that card). Pinned by `cmd/app/main_test.go`, **not** by a component scenario; **not** unit-tested (I/O module, Step 8.1).

## 4. Unit-test formula (Step 8.1)

`N = 1 (happy) + Σ (antecedent branches that yield a distinguishable consequent)`. Head, I/O
modules, and adapters are **not** unit-tested (`module-tree.md` §3, Step 8.1 hard rule).

| Logic module | Happy | Branches | N |
|---|---|---|---|
| `NewConfig` | 1 | empty `consumer.name`; empty `consumed_contract_path`; empty `channels`; duplicate in `channels`; empty-string element in `channels`; empty `provider.name`; both `spec_path`+`spec_url`; neither set; `log_level` out of enum; `timeout <= 0` | 10 | 11 |
| `BuildContractParser` | 1 | none — binds an already-validated scalar, no antecedent branch | 0 | 1 |
| `ContractParser.Parse` | 1 | undecodable content; `schema_version != "1.0"`; `consumer != expectedConsumer`; `captured_hash` regex mismatch; empty `channels`; channel missing `address`/`protocol`; channel with neither `sends` nor `receives` | 7 | 8 |
| `InlineServerRefs` | 1 | (no-op when no `#/servers/*` ref present — same consequent shape, not a branch) | 0 | 1 |
| `DeriveProviderChannels` | 1 | operation with no `messages` → expands to all channel messages; operation with `reply` → injects opposite direction on the reply channel | 2 | 3 |
| `NewComparison` | 1 | a `in.Config.Consumer.Channels` address absent from `in.Contract.Channels` | 1 | 2 |
| `ResolveChannelDirection` | 1 | R1 channel absent; R2 protocol mismatch; R3 direction absent; R4 message key absent | 4 | 5 |
| `CompareMessage` | 1 | R5 missing required sent field; R6 read field not provided; R7 type mismatch; R7 format mismatch; R7 recursion (nested object / array `items` / resolved `$ref`); R8 content-type mismatch (declared); R8 not checked (undeclared); R9 correlationId mismatch (declared); R9 not checked (undeclared) | 9 | 10 |
| `CompareContracts` | 1 | violations accumulate across ≥2 channels without short-circuit (S2 regression anchor); `uncovered_channels` populated without affecting `compatible` (S0 regression anchor) | 2 | 3 |
| `BuildReporter` | 1 | none — binds the clock port, no antecedent branch | 0 | 1 |
| `Reporter.Fold` | 1 | `compatible ⇔ errors == []` both directions | 1 | 2 |
| `BuildSpecLoader` | 1 | selects `FileSpecLoader` vs `HTTPSpecLoader` (1 branch, both outcomes distinguishable) | 1 | 2 |
| `BuildReportWriter` | 1 | `save_json_report == false` → no-op writer (distinguishable behavior at `.Write`, but the *construction* branch itself is the unit) | 1 | 2 |

Total unit tests (this slice): **51**. `R1–R9` are covered **here**, one rule at a time
(`TASK.md` DoD: "Юнит-тестами покрыто ядро: R1–R9 поштучно…") — not as component scenarios (§6).

The two branchless factory rows (`BuildContractParser`, `BuildReporter`, N=1 each) follow ADR-001:
a factory and its product-method are two nodes, and a *logic* factory carries its own row exactly as
`BuildSpecLoader` (N=2) and `BuildReportWriter` (N=2) do. This is **no** carve-out of the
"head/adapters/I/O are not unit-tested" rule: no adapter row, no head row, no I/O row. The two
binary-level Go tests around the reshape (`cmd/app/main_test.go` cases, the `ProcessValidate`
determinism anchor) are deliberately **not** §4 rows — the determinism anchor carries the documented
non-formula marker.

## 5. Gherkin ↔ module reconciliation (Step 8.4)

One row per distinguishable outcome the component suite (§6) asserts.

| Scenario | Then-step | Provided by |
|---|---|---|
| happy: compatible pair | `compatible: true`, `errors: []`, exit 0 | `CompareContracts` (no violations) → `Reporter.Fold` → `cli.ResolveExitCode` |
| happy: compatible pair | `uncovered_channels` lists provider channels outside scope, verdict unaffected | `CompareContracts` (uncovered computation) |
| happy: compatible pair | report file written when `save_json_report` | `ReportWriter.Write` (Success branch) |
| happy: compatible pair | stdout bytes equal `fixtures/validate/good/report.baseline.json` — full byte equality, with only `generated_at`'s *value* normalized identically on both sides (it must be present and parse as RFC3339) | `Reporter.Fold` (canon-1.1 shape, key order, indentation) + `Report.WriteJSON` |
| incompatible: exit-1 verdict | exit 1, stdout is a schema-valid canon-1.1 report, `compatible == false` | `CompareContracts` (violations ≠ ∅) → `Reporter.Fold` → `cli.ResolveExitCode` (grid row 2) |
| incompatible: exit-1 verdict | `errors[0].code == "CHANNEL_NOT_IN_PROVIDER"` | `ResolveChannelDirection` (R1) |
| `CONFIG_ERROR` | exit 2, **no report file/stdout report body** | `ConfigStore.Load`/`NewConfig`/`NewComparison` (Failure: `ErrConfigInvalid`) → `cli.ResolveExitCode` maps to exit 2 |
| `FILE_NOT_FOUND` (consumed-contract) | exit 3, `errors[0].code == FILE_NOT_FOUND`, `subject` = contract path | `ContractStore.Load` (Failure) → `cli.ResolveExitCode` |
| `FILE_NOT_FOUND` (provider spec, file) | exit 3, `errors[0].code == FILE_NOT_FOUND`, `subject` = `spec_path` | `FileSpecLoader.Load` (Failure) → `cli.ResolveExitCode` |
| `PARSE_ERROR` (consumed-contract) | exit 3, `errors[0].code == PARSE_ERROR`, `subject` = contract path | `ContractParser.Parse` (Failure) → `cli.ResolveExitCode` |
| `PARSE_ERROR` (provider spec) | exit 3, `errors[0].code == PARSE_ERROR`, `subject` = spec source | `FileSpecLoader.Load`/`HTTPSpecLoader.Load` (Failure) → `cli.ResolveExitCode` |
| `HTTP_ERROR` | exit 3, `errors[0].code == HTTP_ERROR`, `subject` = `spec_url` | `HTTPSpecLoader.Load` (Failure) → `cli.ResolveExitCode` |
| `TIMEOUT_ERROR` | exit 3, `errors[0].code == TIMEOUT_ERROR`, `subject` = `spec_url` | `HTTPSpecLoader.Load` (Failure) → `cli.ResolveExitCode` |

All Then-steps tie to a named node; checklist items 1–4 (Step 8.5) closed.
`[x] Gherkin-mapping reconciled`.

### The `cli.ResolveExitCode` grid, proven 7/7 through the binary

Recorded here so a reviewer can check it mechanically. The binary path **is** production, so "every
error wrapped exactly as production raises it" holds by construction — strictly stronger than a
hand-built unit grid, and it is why no adapter unit test is needed.

| # | Grid row (`module-tree.md` §5) | Error as production raises it | Proven by |
|---|---|---|---|
| 1 | `err == nil && report.Compatible` → **0** | — | `validate.feature` scenario 1 (happy) |
| 2 | `err == nil && !report.Compatible` → **1** | — | `validate.feature` scenario 9 (incompatible verdict) |
| 3 | `errors.Is(err, ErrConfigInvalid)` → **2** | `fmt.Errorf("%w: …", ErrConfigInvalid)` | `validate.feature` scenario 2 (2a) **+** `main_test.go` `TestValidateCmd_ConfigFileNotFound` |
| 4 | `default:` → **3**, sentinel `FILE_NOT_FOUND` | wrapped `ErrFileNotFound` (`io_contract.go`, `io_spec.go`) | scenarios 3a (contract) and 4a (spec) |
| 5 | `default:` → **3**, sentinel `PARSE_ERROR` | wrapped `ErrParseError` (`logic.go` parser, `io_spec.go`) | scenarios 3b (contract) and 4b (spec) |
| 6 | `default:` → **3**, sentinel `HTTP_ERROR` | wrapped `ErrHTTPError` (`io_spec.go`) | scenario 4c |
| 7 | `default:` → **3**, sentinel `TIMEOUT_ERROR` | wrapped `ErrTimeoutError` (`io_spec.go`) | scenario 4d |

**Totality beyond the grid:** an error matching *no* sentinel — the reachable `ReportWriter.Write`
disk failure — also lands on `default:` → **3**. It is proven by a `cmd/app/main_test.go` case,
deliberately **not** by a scenario (see §6, anti-gaming boundary). Grid coverage: **7/7 through the
binary, with zero adapter unit tests**.

## 6. Component-scenario set (Step 8.6 / `component-tests` DESIGN half)

### Formula and the R1–R9 boundary (why 9, not 8 and not 17)

```
N = 2 (happy-class: compatible verdict + incompatible verdict) + 7 (distinguishable adapter branches) = 9
```

The happy *class* has two members because the tool's two **primary verdicts** — compatible (exit 0)
and incompatible (exit 1) — are both success outcomes of the pipe, not adapter failures:
`CompareContracts` returns `incompatible` as a **value**, never an `Error` (`use-case.md`, note after
6a–6i). The incompatible verdict is a **happy-class member, not an 8th adapter branch** — the
adapter-branch count is 7.

Which Extensions count is decided by **kind**, not by letter:

- **Extensions 6a–6i (R1–R9, the domain verdict)** are **not** an external-dependency failure —
  they are `CompareMessage`/`ResolveChannelDirection`, pure logic over already-valid,
  already-parsed data. Per the `component-tests` skill's boundary rule and `TASK.md`'s own DoD
  ("Юнит-тестами покрыто ядро: R1–R9 поштучно…" / "Компонентными сценариями… покрыты happy-path и
  каждая ветка отказа **адаптера**: `CONFIG_ERROR`, `FILE_NOT_FOUND`, `PARSE_ERROR`, `HTTP_ERROR`,
  `TIMEOUT_ERROR`"), these nine Extensions map to **unit boundaries** (§4), not component
  scenarios. Scenario 9 does **not** reopen this: it asserts the *verdict and its exit code*
  end-to-end via one representative rule (R1), exactly as scenario 1 asserts the compatible verdict
  end-to-end — the granular rule-by-rule proof stays the units' job. Nine scenarios, one per rule,
  would be the violation.
- **Extensions 2a, 3a, 3b, 4a, 4b, 4c, 4d** are genuine I/O-adapter failures (config/contract/spec
  file or HTTP access). These **do** count — 7 of them.
- Within that set, `use-case.md`'s own traceability note applies: 3a/4a share the
  `FILE_NOT_FOUND` string and 3b/4b share `PARSE_ERROR`, but **each is still its own scenario**
  because the failing adapter/fixture differs (consumed-contract vs. provider spec); the branch
  count is by (adapter, fixture), not by code string.

Fixtures for scenarios 1–8 exist as a dry trace in
[`../../../sandbox/EMULATION.md`](../../../sandbox/EMULATION.md) (S0/S11–S15, S12/S3a-equivalent,
S13/S4b-equivalent — see the table below for the exact mapping).

### Scenario table (Cockburn wording verbatim)

| # | Scenario name (verbatim from `use-case.md`) | Class / adapter branch | Fixture source | Exit | `errors[0].code` |
|---|---|---|---|---|---|
| 1 | Happy path: consumer compatible with provider | happy-class (compatible verdict) | EMULATION.md S0, `fixtures/validate/good/` | 0 | — (`errors: []`) |
| 2 | 2a. Config fails schema validation | `ConfigStore.Load`/`NewConfig`/`NewComparison` | EMULATION.md S11 (`spec_path`+`spec_url` both set); representative fixture for the schema-breach class | 2 | — (no report) |
| 3 | 3a. `consumed_contract_path` is unreachable at run time | `ContractStore.Load` | EMULATION.md S12 | 3 | `FILE_NOT_FOUND` |
| 4 | 3b. `consumed-contract` fails to parse or validate | `ContractParser.Parse` | malformed/consumer-mismatch consumed-contract fixture | 3 | `PARSE_ERROR` |
| 5 | 4a. `spec_path` is configured but the file is unreachable | `FileSpecLoader.Load` | missing spec file fixture | 3 | `FILE_NOT_FOUND` |
| 6 | 4b. The provider spec does not parse as AsyncAPI 3.0 | `FileSpecLoader.Load`/`HTTPSpecLoader.Load` | EMULATION.md S13 | 3 | `PARSE_ERROR` |
| 7 | 4c. `spec_url` is unreachable or returns a non-2xx status | `HTTPSpecLoader.Load` | EMULATION.md S14 | 3 | `HTTP_ERROR` |
| 8 | 4d. The `spec_url` request exceeds `settings.timeout` | `HTTPSpecLoader.Load` | EMULATION.md S15 | 3 | `TIMEOUT_ERROR` |
| 9 | Consumer incompatible with provider (primary verdict) | happy-class (incompatible verdict) — NOT an adapter branch | `fixtures/validate/incompatible/`: `good/` + one channel address added to BOTH `consumer.channels` AND the consumed-contract's `channels`, absent from the provider spec → R1 | 1 | `CHANNEL_NOT_IN_PROVIDER` |

**Tag discipline:** all nine scenarios are accepted and untagged; `@wip` marks new surface only until
`@fagan` strips it at slice acceptance.

**Assertion order for scenario 9:** exit code **1**; stdout is a schema-valid canon-`1.1` report
(`report.schema.json`); `compatible == false`; `errors[0].code == "CHANNEL_NOT_IN_PROVIDER"`.

**Fixture trap (repeated because it silently proves the wrong branch):** the added address MUST be
present in the consumed-contract. Absent there, `NewComparison`'s cross-artifact check fires →
`ErrConfigInvalid` → exit **2**, and the scenario would pass for the wrong reason. `good/`'s three
files MUST NOT be modified (the stdout byte baseline depends on their bytes).

### Gate check (Step 8.6)

```
#component_adapter_failure_scenarios (7) == #distinguishable_adapter_branches (7)
```

The 9 verdict Extensions (6a–6i) have their unit boundaries in §4 instead and are not counted here.
Total scenarios = 7 adapter + 2 happy-class = **9**.

**Anti-gaming boundary — MUST NOT add a 10th scenario.** The out-of-taxonomy `ReportWriter.Write`
failure is **not** one of the 7 adapter branches; making it a scenario would break the gate line
above. It lives in `cmd/app/main_test.go` for exactly that reason, and it asserts **only**
`*exitError` with `code == 3` — the stdout body on that path is schema-invalid today and pinning it
requires a frozen-contract decision (recorded as debt).

`component-tests/features/validate.feature` carries the formula above and a STOP-warning against a
**10th** scenario; `node harness/validate-component-tests.mjs .` must exit 0 and report **9**
business scenarios.

<!-- DONE: contracts slice-01-validate -->
<!-- DONE: moduledesigner slice-01-validate -->
