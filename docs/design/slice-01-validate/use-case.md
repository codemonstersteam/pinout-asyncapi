# UC-1: Validate consumer↔provider compatibility (async, AsyncAPI 3.0)

> Fully-dressed expansion of FRD § 3, UC1 ("Validate consumer↔provider compatibility") for slice
> **slice-01-validate** (package `internal/validate/`). Behavioral source of truth; the wording of
> each Extension is reused verbatim as the matching component-test scenario name. The cross-artifact
> traceability key is the **`error.code`** (stable FRD ↔ use-case ↔ contract), not the Cockburn
> `a/b/c` label — sync-sibling `pinout-openapi`'s `docs/design/slice-01-validate/use-case.md` is the
> format reference.

- **Primary actor**: Consumer CI pipeline (pre-merge gate). Secondary: consumer developer running
  the binary locally.
- **Scope**: `pinout-asyncapi` — Compatibility Validation (single bounded context).
- **Level**: user-goal (sea level).
- **Stakeholders & interests**:
  - **Consumer CI** — wants a deterministic, machine-readable verdict it can gate a merge on.
  - **Consumer team** — does not want a false `incompatible` (blocked merge on a non-issue) and does
    not want a missed real compatibility break.
  - **Provider team** — bears no validation duty; its spec is read-only ground truth, never mutated
    or called.
  - **`pinout-netlist`** — consumes the report as a typed artifact for its dependency graph and
    provenance tracking; expects an exception-free canon-`1.1` report.
  - **pinout ecosystem operator (E0/E1)** — expects symmetry with `pinout-openapi` (config, report,
    exit codes) with no undocumented divergence.
- **Precondition**: the `pinout-asyncapi` binary is installed and executable in the CI/developer
  environment; `<config.yaml>` is a path passed in argv (the path itself may be unreadable — that is
  an Extension, not a precondition failure); `PINOUT_PROVIDER_TOKEN`, if a private `spec_url`
  requires it, is available in the process environment (never in the config, never in git).
- **Trigger**: the consumer CI invokes `pinout-asyncapi validate <config.yaml>` (one-shot CLI
  invocation — the slice's single external input).
- **Minimal guarantee**: on any outcome (Main Success Scenario or any Extension) exactly one
  machine-readable exit code (0/1/2/3) is set and the process terminates; `PINOUT_PROVIDER_TOKEN`
  never appears in the report or in logs; on `exit ∈ {2}` no report is written; on `exit ∈ {3}` a
  report may carry the io/parse code in `errors[]` (§7 failure-mode map).
- **Success guarantee (postcondition)**: exactly one canon-`1.1` JSON report is printed to stdout
  (plus written to `settings.json_report_file` when `settings.save_json_report`); the report is
  valid against `report.schema.json` whenever `exit ∈ {0,1}`; the aggregate verdict is
  `compatible ⇔ errors == []`; the same input bytes (config + consumed-contract + provider spec)
  under the same `PINOUT_PROVIDER_TOKEN` yield the same verdict and the same report bytes except
  `generated_at` (injected by the clock port, NFR §8).

## Main Success Scenario

1. The **consumer CI** invokes `pinout-asyncapi validate <config.yaml>`.
2. The system reads `config.yaml` and validates it against `config.schema.json`, producing `Config`.
3. The system reads the consumer's `consumed-contract` (path from `Config`) and validates it against
   `consumed-contract.schema.json`, producing `ConsumedContract`.
4. The system obtains the provider's spec (`spec_path` XOR `spec_url` from `Config`; for `spec_url`,
   an HTTP GET with `Authorization: Bearer $PINOUT_PROVIDER_TOKEN` from env) and parses it as
   AsyncAPI 3.0, pre-resolving `#/servers/*` refs first (`InlineServerRefs`, a parser-defect
   workaround, D2).
5. The system projects the provider spec into `{address → protocol, messages per direction}`
   (`DeriveProviderChannels`), expanding two AsyncAPI defaults: an operation with no `messages` means
   all of the channel's messages, and `reply` (D3).
6. The system compares `ConsumedContract` against the provider projection per rules R1–R9 (FRD § 6),
   for every channel in `consumer.channels`, over both directions, over `payload` and `headers`,
   accumulating every violation without short-circuiting (several may surface in one run).
7. The system folds the outcome into a canon-`1.1` report: `compatible` (⇔ `errors == []`),
   `errors[]`, `provenance` (echoed from `ConsumedContract`), `uncovered_channels[]`, `generated_at`
   (injected by the clock port — the core never reads the system clock).
8. The system prints the report to stdout; when `settings.save_json_report` (default `true`), also
   writes it to `settings.json_report_file`.
9. The system exits with code **0**, since `compatible == true`.

## Extensions

*Each Extension = one failure-mode-map row (FRD § 7) = one `error.code`. Two Extensions
(3a/4a and 3b/4b) share an `error.code` string across two distinguishable artifacts (consumed-contract
vs. provider spec) — each is still its own row/scenario because the triggering artifact and the
component-test fixture differ. Counts: 16 Extensions == 16 failure-mode rows == 14 distinct
`error.code` values (`CONFIG_ERROR` + `FILE_NOT_FOUND` + `PARSE_ERROR` + `HTTP_ERROR` +
`TIMEOUT_ERROR` + R1–R9), matching FRD § 7 ("13 distinct codes in `errors[]`, `CONFIG_ERROR` a 14th
state outside `errors[]`").*

- **2a. `Config` fails schema validation** (step 2) — any `config.schema.json` breach: empty/missing
  required field, both or neither of `spec_path`/`spec_url` present, an empty/duplicate/empty-string
  element in `consumer.channels`, a `consumer.channels` entry absent from the `consumed-contract`, a
  `settings` value out of range/enum: the system rejects the config before any comparison and before
  any report is written → `error.code = CONFIG_ERROR`, exit **2** *(FRD Extension 2a)*.
- **3a. `consumed_contract_path` is unreachable at run time** (step 3) — file missing or unreadable:
  the system reports the missing input artifact → `error.code = FILE_NOT_FOUND`, exit **3**
  *(FRD Extension 3a)*.
- **3b. `consumed-contract` fails to parse or validate** (step 3) — malformed content, schema breach,
  `schema_version ≠ "1.0"`, or the contract's `consumer` field does not match `config.consumer.name`
  verbatim: the system reports the parse/validation failure → `error.code = PARSE_ERROR`, exit **3**
  *(FRD Extension 3b)*.
- **4a. `spec_path` is configured but the file is unreachable** (step 4): the system reports the
  missing input artifact → `error.code = FILE_NOT_FOUND`, exit **3** *(FRD Extension 4a)*.
- **4b. The provider spec (file or HTTP response body) does not parse as AsyncAPI 3.0** (step 4): the
  system reports the parse failure → `error.code = PARSE_ERROR`, exit **3** *(FRD Extension 4b)*.
- **4c. `spec_url` is configured but the HTTP request is unreachable or returns a non-2xx status**
  (step 4): the system reports the fetch failure → `error.code = HTTP_ERROR`, exit **3**
  *(FRD Extension 4c)*.
- **4d. The `spec_url` request exceeds `settings.timeout`** (step 4): the system aborts the fetch and
  reports the timeout → `error.code = TIMEOUT_ERROR`, exit **3** *(FRD Extension 4d)*.
- **6a. A channel address from `consumer.channels` is absent from the provider** (step 6, R1): the
  system records the violation → `error.code = CHANNEL_NOT_IN_PROVIDER`, verdict `incompatible`,
  exit **1** *(FRD Extension 6a, rule R1)*.
- **6b. The contract's `channel.protocol` does not match the provider channel's server protocol**
  (step 6, R2): the system records the violation → `error.code = PROTOCOL_MISMATCH`, verdict
  `incompatible`, exit **1** *(FRD Extension 6a, rule R2)*.
- **6c. The provider has no counterpart operation for the consumer's direction on a channel**
  (step 6, R3, `reply` expanded per D3): the system records the violation →
  `error.code = DIRECTION_NOT_IN_PROVIDER`, verdict `incompatible`, exit **1**
  *(FRD Extension 6a, rule R3)*.
- **6d. The direction exists but the contract's message key is absent from the provider's
  `channel.messages` on that direction** (step 6, R4): the system records the violation →
  `error.code = MESSAGE_NOT_IN_PROVIDER`, verdict `incompatible`, exit **1**
  *(FRD Extension 6a, rule R4)*.
- **6e. On a `sends` message, a field the provider requires is not sent by the consumer**
  (step 6, R5, contravariant `required(provider) ⊆ fields(consumer.sends)`): the system records the
  violation → `error.code = MISSING_REQUIRED_SENT_FIELD`, verdict `incompatible`, exit **1**
  *(FRD Extension 6a, rule R5)*.
- **6f. On a `receives` message, the consumer reads a field the provider does not offer**
  (step 6, R6, covariant `fields(consumer.receives) ⊆ properties(provider)` — catches provider field
  removal): the system records the violation → `error.code = READS_FIELD_NOT_PROVIDED`, verdict
  `incompatible`, exit **1** *(FRD Extension 6a, rule R6)*.
- **6g. A shared field's `type`/`format` diverges between consumer and provider, recursively, in
  `payload` or `headers`** (step 6, R7): the system records the violation →
  `error.code = TYPE_MISMATCH`, verdict `incompatible`, exit **1** *(FRD Extension 6a, rule R7)*.
- **6h. The message's effective `contentType` diverges, when `message.content_type` is declared in
  the contract** (step 6, R8): the system records the violation →
  `error.code = CONTENT_TYPE_MISMATCH`, verdict `incompatible`, exit **1**
  *(FRD Extension 6a, rule R8)*.
- **6i. The message's `correlationId.location` diverges, when `message.correlation_id_location` is
  declared in the contract** (step 6, R9, not gated by channel communication pattern, D9): the
  system records the violation → `error.code = CORRELATION_ID_MISMATCH`, verdict `incompatible`,
  exit **1** *(FRD Extension 6a, rule R9)*.

> Extensions **6a–6i** are a *verdict* — `incompatible` is a legitimate domain answer (MSS continues
> to step 9 normally, just with `compatible = false`), not a tool fault; all nine accumulate across
> channels/directions/fields in one run (step 6) before the verdict is rendered. Extension **2a**
> (config, exit 2) is detected before any report exists. Extensions **3a/3b/4a/4b/4c/4d** (io/parse
> of the two input artifacts — consumed-contract and provider spec) are tool errors on the input
> side, exit 3.

## Technology & data variations

- **Provider spec source** (step 4): `spec_path` (local file) XOR `spec_url` (HTTP GET). A private
  `spec_url` is fetched with `Authorization: Bearer $PINOUT_PROVIDER_TOKEN` sourced from env; public
  URLs work without a token. `spec_path` resolves relative to the config file's directory;
  `json_report_file` resolves relative to the process working directory (both fixed by schema, an
  intentional asymmetry, not a contradiction).
- **Uncovered provider surface** (step 7): provider channels outside `consumer.channels` are listed
  in the report as `uncovered_channels[]` — informational only, no verdict or exit-code effect.
- **Parser limitation** (step 4): `github.com/lerenn/asyncapi-codegen` (pinned `v0.63.0`) does not
  support external file `$ref`s and does not expand `allOf`/`anyOf`/`oneOf`; protocol bindings are
  not compared directly (only indirectly, via R2 at channel level).
- **Message identity** (step 6): messages are matched by the map key of the provider spec's
  `channel.messages`, not by `Message.Name` (the parser overwrites `Name` with a Go identifier, D8).

<!-- DONE: usecase slice-01-validate -->
