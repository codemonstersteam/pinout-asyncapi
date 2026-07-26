# slice-01-validate — module contracts, message catalog, tests, component scenarios

> Companion to [`module-tree.md`](./module-tree.md) (Steps 4/5/6/8 of `program-design`). Frozen
> contracts referenced throughout: [`config.schema.json`](../../../api-specification/config.schema.json),
> [`consumed-contract.schema.json`](../../../api-specification/consumed-contract.schema.json),
> [`report.schema.json`](../../../api-specification/report.schema.json).

## 1. Message catalog (Step 4)

| Type | Kind | Notes |
|---|---|---|
| `Invocation` | `Request` | `{ConfigPath string}` — unvalidated, from `cli.Parse`. |
| `RawConfig` | raw | decoded YAML/JSON bytes of `config.yaml`, pre-validation. |
| `Config` | `Entity` | valid-by-construction; unexported fields; built only via `NewConfig`. |
| `RawContract` | raw | decoded bytes of the `consumed-contract` artifact, pre-validation. |
| `ConsumedContract` | `Entity` | valid-by-construction; built only via `NewConsumedContract`. |
| `ProviderSpec` | `Entity` (external) | the `lerenn/asyncapi-codegen` typed AsyncAPI 3.0 model, post `InlineServerRefs` + `Process()`. |
| `ProviderChannels` | DTO | `map[address]{protocol, send: []MessageRef, receive: []MessageRef}` — `DeriveProviderChannels`'s output. |
| `Comparison` | `Entity` | valid-by-construction; unites `Config` + `ConsumedContract` + `ProviderChannels`; built only via `NewComparison`. |
| `Outcome` | DTO | `{Violations []Violation, UncoveredChannels []string, ConsumerName string, Provenance Provenance}` — `CompareContracts`'s output; carries forward everything `FoldReport` needs besides the clock. |
| `Report` | DTO | canon-`1.1` shape, `report.schema.json`. |
| `Error` | `Error` | see §2. |

**Signature rule (Go idiom):** every `Result<T, Error>` above is a Go `(T, error)` return pair.

## 2. Error model

| `error.code` | Raised by | Exit | In `errors[]`? |
|---|---|---|---|
| `CONFIG_ERROR` | `ConfigStore.Load` (unreadable), `NewConfig` (schema breach), `NewComparison` (channel scope not in consumed-contract) | 2 | **no** — detected before `FoldReport` runs |
| `FILE_NOT_FOUND` | `ContractStore.Load`; `FileSpecLoader.Load` | 3 | yes |
| `PARSE_ERROR` | `NewConsumedContract`; `FileSpecLoader.Load`/`HTTPSpecLoader.Load` (body doesn't parse as AsyncAPI 3.0) | 3 | yes |
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
module (io/parse codes, subject = the artifact path/URL); `FoldReport` never recomputes it.

## 3. Module contracts

Only behavioral modules get a card; `domain.go`/`errors.go` (types/sentinels) and `register.go`
(wiring) do not (`program-design` Step 5).

### `cli.Parse`
- **Signature:** `Parse(args []string) -> Result<Invocation, Error>`
- **Input (data):** `args []string` (argv)
- **Dependencies:** —
- **io:** `none`
- **What it does:** parses the one positional arg into the flat `Request`.
- **Antecedent:** exactly one non-flag argument present (else usage error, not a domain `Error` — cobra's own `--help`/usage path).
- **Consequent:** success → `Invocation{ConfigPath}`. No domain failure branch (parsing only).

### `ProcessValidate`
- **Signature:** `ProcessValidate(inv: Invocation, deps: Deps) -> Result<Report, Error>`
- **Input (data):** `Invocation`
- **Dependencies:** `Deps{Clock clock.Clock, HTTPClient *http.Client}`
- **io:** `none`
- **What it does:** orchestrates the pipe (module-tree.md §4); no logic of its own.
- **Antecedent:** a well-formed `Invocation`.
- **Consequent:** success → `Report`; failure → the first short-circuiting step's `Error`, untransformed.

### `cli.ResolveExitCode`
- **Signature:** `ResolveExitCode(res: Result<Report, Error>) -> int`
- **Input (data):** `Result<Report, Error>`
- **Dependencies:** —
- **io:** `none`
- **What it does:** maps the outcome to the exit-code grid (§5 in `module-tree.md`); prints the report to stdout on any branch that has one, logs to stderr.
- **Antecedent:** none (total function over the closed `Error` taxonomy).
- **Consequent:** exactly one of `{0,1,2,3}`.

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

### `NewConsumedContract`
- **Signature:** `NewConsumedContract(raw: RawContract) -> Result<ConsumedContract, Error>`
- **Input (data):** `RawContract`
- **Dependencies:** `expectedConsumer string` (= `cfg.Consumer.Name`, a config value, not I/O)
- **io:** `none`
- **What it does:** validates against `consumed-contract.schema.json`; builds the valid-by-construction `ConsumedContract`.
- **Antecedent:** decodes as valid YAML/JSON; `schema_version == "1.0"`; `consumer == expectedConsumer`; `provenance.captured_hash` matches `^sha256:[0-9a-f]{64}$`; `channels` non-empty; each channel has non-empty `address`/`protocol` and at least one of `sends`/`receives`.
- **Consequent:** success → `ConsumedContract`; failure (any antecedent clause) → `ErrContractInvalid` (`PARSE_ERROR`).

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
- **Signature:** `NewComparison(cfg: Config, contract: ConsumedContract, pchans: ProviderChannels) -> Result<Comparison, Error>`
- **Input (data):** `Config`, `ConsumedContract`, `ProviderChannels` — **the uniting-constructor exception** (module-tree.md §3)
- **Dependencies:** —
- **io:** `none`
- **What it does:** unites the three already-valid parts into one valid `Comparison`; performs the one cross-artifact check that can only run here (FRD §4.1: every address in `cfg.Consumer.Channels` must appear in `contract.Channels`).
- **Antecedent:** every address in `cfg.Consumer.Channels` is present among `contract.Channels[].Address`.
- **Consequent:** success → `Comparison`; failure → `ErrConfigInvalid` (`CONFIG_ERROR`) — still before any report is written.

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
- **What it does:** folds `ResolveChannelDirection` + `CompareMessage` over every channel (sorted order, determinism) in `consumer.channels`, accumulating violations without short-circuiting across channels; also computes `uncovered_channels[]` (provider channels outside `consumer.channels`) and carries forward `ConsumerName`/`Provenance` for `FoldReport`.
- **Antecedent:** a valid `Comparison`.
- **Consequent:** `Outcome{Violations, UncoveredChannels, ConsumerName, Provenance}` — never an `Error`; `incompatible` is a value, not a failure (use-case.md, note after 6a–6i).

### `FoldReport`
- **Signature:** `FoldReport(outcome: Outcome) -> Report`
- **Input (data):** `Outcome`
- **Dependencies:** `clock.Clock` (calls `.Now()` exactly once — the whole slice's only clock read, D10)
- **io:** `none`
- **What it does:** assembles the canon-`1.1` `Report`: `compatible ⇔ errors == []`, `errors[]` (from `Violations`), `provenance` (echoed), `uncovered_channels[]`, `generated_at` (from the injected clock), `validator`/`interaction` build-time constants.
- **Antecedent:** a valid `Outcome`.
- **Consequent:** `Report`, total — no failure branch.

### `BuildReportWriter`
- **Signature:** `BuildReportWriter(settings: Settings) -> ReportWriter`
- **Input (data):** `Settings` (`save_json_report`, `json_report_file`)
- **Dependencies:** —
- **io:** `none`
- **What it does:** constructs the writer, encapsulating whether/where to persist (Store-style: the head never branches on `save_json_report` itself).
- **Antecedent:** —
- **Consequent:** a `ReportWriter` (no-op internally when `save_json_report == false`).

### `ReportWriter.Write`
- **Signature:** `(w ReportWriter) Write(report: Report) -> Result<Report, Error>`
- **Input (data):** `Report`
- **Dependencies:** —
- **io:** `none` (filesystem pipe)
- **What it does:** writes the report bytes to the configured path (or no-op, per construction); always returns the report unchanged for the pipe to continue.
- **Antecedent:** —
- **Consequent:** success → `Report` (pass-through). A disk-write failure here has **no code in the frozen error taxonomy** (`report.schema.json` `x-exit-codes` has none for it) — out of scope for this design; flagged for the ticket-writing stage as an implementation note, not a new component scenario (frozen contracts are not modified by this stage).

## 4. Unit-test formula (Step 8.1)

`N = 1 (happy) + Σ (antecedent branches that yield a distinguishable consequent)`. Head, I/O
modules, and adapters are **not** unit-tested (module-tree.md §3, Step 8.1 hard rule).

| Logic module | Happy | Branches | N |
|---|---|---|---|
| `NewConfig` | 1 | empty `consumer.name`; empty `consumed_contract_path`; empty `channels`; duplicate in `channels`; empty-string element in `channels`; empty `provider.name`; both `spec_path`+`spec_url`; neither set; `log_level` out of enum; `timeout <= 0` | 10 | 11 |
| `NewConsumedContract` | 1 | undecodable content; `schema_version != "1.0"`; `consumer != expectedConsumer`; `captured_hash` regex mismatch; empty `channels`; channel missing `address`/`protocol`; channel with neither `sends` nor `receives` | 7 | 8 |
| `InlineServerRefs` | 1 | (no-op when no `#/servers/*` ref present — same consequent shape, not a branch) | 0 | 1 |
| `DeriveProviderChannels` | 1 | operation with no `messages` → expands to all channel messages; operation with `reply` → injects opposite direction on the reply channel | 2 | 3 |
| `NewComparison` | 1 | a `cfg.Consumer.Channels` address absent from `contract.Channels` | 1 | 2 |
| `ResolveChannelDirection` | 1 | R1 channel absent; R2 protocol mismatch; R3 direction absent; R4 message key absent | 4 | 5 |
| `CompareMessage` | 1 | R5 missing required sent field; R6 read field not provided; R7 type mismatch; R7 format mismatch; R7 recursion (nested object / array `items` / resolved `$ref`); R8 content-type mismatch (declared); R8 not checked (undeclared); R9 correlationId mismatch (declared); R9 not checked (undeclared) | 9 | 10 |
| `CompareContracts` | 1 | violations accumulate across ≥2 channels without short-circuit (S2 regression anchor); `uncovered_channels` populated without affecting `compatible` (S0 regression anchor) | 2 | 3 |
| `FoldReport` | 1 | `compatible ⇔ errors == []` both directions | 1 | 2 |
| `BuildSpecLoader` | 1 | selects `FileSpecLoader` vs `HTTPSpecLoader` (1 branch, both outcomes distinguishable) | 1 | 2 |
| `BuildReportWriter` | 1 | `save_json_report == false` → no-op writer (distinguishable behavior at `.Write`, but the *construction* branch itself is the unit) | 1 | 2 |

Total unit tests (this slice): **49**. `R1–R9` are covered **here**, one rule at a time
(`TASK.md` DoD: "Юнит-тестами покрыто ядро: R1–R9 поштучно…") — not as component scenarios (§6).

## 5. Gherkin ↔ module reconciliation (Step 8.4)

One row per distinguishable outcome the component suite (§6) asserts:

| Scenario | Then-step | Provided by |
|---|---|---|
| happy: compatible pair | `compatible: true`, `errors: []`, exit 0 | `CompareContracts` (no violations) → `FoldReport` → `cli.ResolveExitCode` |
| happy: compatible pair | `uncovered_channels` lists provider channels outside scope, verdict unaffected | `CompareContracts` (uncovered computation) |
| happy: compatible pair | report file written when `save_json_report` | `ReportWriter.Write` (Success branch) |
| `CONFIG_ERROR` | exit 2, **no report file/stdout report body** | `ConfigStore.Load`/`NewConfig`/`NewComparison` (Failure: `ErrConfigInvalid`) → `cli.ResolveExitCode` maps to exit 2 |
| `FILE_NOT_FOUND` (consumed-contract) | exit 3, `errors[0].code == FILE_NOT_FOUND`, `subject` = contract path | `ContractStore.Load` (Failure) → `cli.ResolveExitCode` |
| `FILE_NOT_FOUND` (provider spec, file) | exit 3, `errors[0].code == FILE_NOT_FOUND`, `subject` = `spec_path` | `FileSpecLoader.Load` (Failure) → `cli.ResolveExitCode` |
| `PARSE_ERROR` (consumed-contract) | exit 3, `errors[0].code == PARSE_ERROR`, `subject` = contract path | `NewConsumedContract` (Failure) → `cli.ResolveExitCode` |
| `PARSE_ERROR` (provider spec) | exit 3, `errors[0].code == PARSE_ERROR`, `subject` = spec source | `FileSpecLoader.Load`/`HTTPSpecLoader.Load` (Failure) → `cli.ResolveExitCode` |
| `HTTP_ERROR` | exit 3, `errors[0].code == HTTP_ERROR`, `subject` = `spec_url` | `HTTPSpecLoader.Load` (Failure) → `cli.ResolveExitCode` |
| `TIMEOUT_ERROR` | exit 3, `errors[0].code == TIMEOUT_ERROR`, `subject` = `spec_url` | `HTTPSpecLoader.Load` (Failure) → `cli.ResolveExitCode` |

All Then-steps tie to a named node; checklist items 1–4 (Step 8.5) closed.
`[x] Gherkin-mapping reconciled`.

## 6. Component-scenario set (Step 8.6 / `component-tests` DESIGN half)

### Formula and the R1–R9 boundary (why 8, not 17)

`N = 1 (happy) + Σ (distinguishable I/O-adapter branches)`. Cockburn Extensions give the wording;
which Extensions count is decided by **kind**, not by letter:

- **Extensions 6a–6i (R1–R9, the domain verdict)** are **not** an external-dependency failure —
  they are `CompareMessage`/`ResolveChannelDirection`, pure logic over already-valid,
  already-parsed data. Per the `component-tests` skill's boundary rule and `TASK.md`'s own DoD
  ("Юнит-тестами покрыто ядро: R1–R9 поштучно…" / "Компонентными сценариями… покрыты happy-path и
  каждая ветка отказа **адаптера**: `CONFIG_ERROR`, `FILE_NOT_FOUND`, `PARSE_ERROR`, `HTTP_ERROR`,
  `TIMEOUT_ERROR`"), these nine Extensions map to **unit boundaries** (§4), not component
  scenarios. One component scenario (the happy path) exercises `CompareContracts` end-to-end on
  the success side; that is sufficient black-box proof the wiring works — the granular rule-by-rule
  proof is the units' job.
- **Extensions 2a, 3a, 3b, 4a, 4b, 4c, 4d** are genuine I/O-adapter failures (config/contract/spec
  file or HTTP access). These **do** count.
- Within that set, `use-case.md`'s own traceability note applies: 3a/4a share the
  `FILE_NOT_FOUND` string and 3b/4b share `PARSE_ERROR`, but **each is still its own scenario**
  because the failing adapter/fixture differs (consumed-contract vs. provider spec) — "a scenario
  without an adapter branch, or an adapter branch without a scenario" would be the mismatch to
  fix (Step 8.6 gate); here the branch count is by (adapter, fixture), not by code string.

Count: **7 distinguishable adapter branches** (`CONFIG_ERROR` ×1, `FILE_NOT_FOUND` ×2,
`PARSE_ERROR` ×2, `HTTP_ERROR` ×1, `TIMEOUT_ERROR` ×1) **+ 1 happy = 8 component scenarios**,
matching `TASK.md`'s DoD list of codes plus the artifact-level split `use-case.md` requires.
Fixtures for every scenario already exist as a dry trace in
[`../../../sandbox/EMULATION.md`](../../../sandbox/EMULATION.md) (S0/S11–S15, S12/S3a-equivalent,
S13/S4b-equivalent — see the scenario table below for the exact mapping); the component-test
realize stage (`wirth-tester`) drops these into `.feature` files, not new ones.

### Scenario table (Cockburn wording verbatim; tag `@wip`)

| # | Scenario name (verbatim from `use-case.md`) | Adapter branch | Fixture source | Exit | `errors[0].code` |
|---|---|---|---|---|---|
| 1 | `@wip` Happy path: consumer compatible with provider | — (success) | EMULATION.md S0 | 0 | — (`errors: []`) |
| 2 | `@wip` 2a. Config fails schema validation | `ConfigStore.Load`/`NewConfig`/`NewComparison` | EMULATION.md S11 (`spec_path`+`spec_url` both set); representative fixture for the schema-breach class | 2 | — (no report) |
| 3 | `@wip` 3a. `consumed_contract_path` is unreachable at run time | `ContractStore.Load` | EMULATION.md S12 | 3 | `FILE_NOT_FOUND` |
| 4 | `@wip` 3b. `consumed-contract` fails to parse or validate | `NewConsumedContract` | malformed/consumer-mismatch consumed-contract fixture | 3 | `PARSE_ERROR` |
| 5 | `@wip` 4a. `spec_path` is configured but the file is unreachable | `FileSpecLoader.Load` | missing spec file fixture | 3 | `FILE_NOT_FOUND` |
| 6 | `@wip` 4b. The provider spec does not parse as AsyncAPI 3.0 | `FileSpecLoader.Load`/`HTTPSpecLoader.Load` | EMULATION.md S13 | 3 | `PARSE_ERROR` |
| 7 | `@wip` 4c. `spec_url` is unreachable or returns a non-2xx status | `HTTPSpecLoader.Load` | EMULATION.md S14 | 3 | `HTTP_ERROR` |
| 8 | `@wip` 4d. The `spec_url` request exceeds `settings.timeout` | `HTTPSpecLoader.Load` | EMULATION.md S15 | 3 | `TIMEOUT_ERROR` |

Gate check (Step 8.6): `#component_failure_scenarios (7) == #distinguishable_adapter_branches (7)`;
the 9 verdict Extensions (6a–6i) have their unit boundaries in §4 instead, and are not counted
here. Realize stage: `wirth-tester` (component-tests skill, REALIZE half).

<!-- DONE: contracts slice-01-validate -->
