---
id: 03
type: module
slice: slice-01-validate
blocked_by: [01, 02]
inputs: [docs/design/slice-01-validate/module-tree.md, docs/design/slice-01-validate/contracts.md, api-specification/config.schema.json, api-specification/consumed-contract.schema.json, api-specification/report.schema.json]
outputs: [internal/validate/domain.go, internal/validate/errors.go]
io: none
skills: []
---

### TICKET 03 — slice-01-validate/foundation: shared slice types + sentinel errors (package `validate`)

**io:** none → skills: [] (pure type/sentinel files — carry NO behavioral contract; needed FIRST
by every module, so they are their own foundation ticket, not folded onto a leaf).

**Context (only these two files):**
- `internal/validate/domain.go` — the slice's shared value types (`module-tree.md` §6 layout row;
  message catalog `contracts.md` §1). Field domains come from the frozen schemas — do NOT invent:
  - `Invocation{ConfigPath string}` — unvalidated, from `cli.Parse`.
  - `RawConfig` — decoded YAML/JSON, pre-validation.
  - `Config` — **unexported fields**, built only via `NewConfig`: `Consumer{Name string,
    ConsumedContractPath string, Channels []string}`, `Provider{Name, SpecPath, SpecURL string}`,
    `Settings{LogLevel string, SaveJSONReport bool, JSONReportFile string, Timeout
    time.Duration, IgnoreWarnings bool}` — mirrors `config.schema.json`.
  - `RawContract` — decoded bytes of the consumed-contract artifact, pre-validation.
  - `ConsumedContract` — **unexported fields**, built only via `NewConsumedContract`:
    `Consumer string`, `Provenance Provenance`, `Channels []Channel{Address, Protocol string,
    Sends, Receives []Message}` — mirrors `consumed-contract.schema.json`.
  - `Message{Name, ContentType, CorrelationIDLocation string, Payload, Headers *SchemaNode}`
    (`ContentType`/`CorrelationIDLocation` empty ⇒ "not declared", per D9/R8/R9).
  - `SchemaNode{Type, Format string, Properties map[string]*SchemaNode, Items *SchemaNode}` —
    mirrors the consumed-contract `$defs/schema` (deliberately no `required`/`enum`, per schema).
  - `Provenance{Provider, ProviderVersion, CapturedHash string}`.
  - `ProviderSpec` — thin wrapper around the `lerenn/asyncapi-codegen` typed model returned by
    `parser.FromYAML(...).Process()` (opaque to the rest of the slice beyond `DeriveProviderChannels`).
  - `ProviderChannels map[string]ProviderChannel{Protocol string, Send, Receive []string}` —
    `DeriveProviderChannels`'s projection (message refs = `channel.messages` map keys, D8).
  - `Comparison` — **unexported fields**, built only via `NewComparison`: unites `Config` +
    `ConsumedContract` + `ProviderChannels`.
  - `Violation{Code, Message, Subject, Location, Details string, Context map[string]string}`.
  - `Outcome{Violations []Violation, UncoveredChannels []string, ConsumerName string,
    Provenance Provenance}`.
  - `Report` — canon-`1.1` shape (`report.schema.json`): `SchemaVersion, Validator, Interaction
    string`, `Consumer{Name string, Version string ",omitempty"}`, `GeneratedAt string` (RFC3339),
    `Compatible bool`, `Provenance Provenance`, `Errors []Violation`, `UncoveredChannels []string`.
  - **`Clock` port** — `type Clock interface { Now() time.Time }`. Declared HERE (not in
    `head.go`) so `FoldReport` (a `logic.go` function, ticket 19) and `head.go`'s `Deps`
    (ticket 22) both see it without a forward reference — a deliberate, low-risk placement choice
    within this file layout's discretion (`contracts.md`: only behavioral modules get a card;
    `domain.go` does not, so its internal split is not frozen).
- `internal/validate/errors.go` — sentinel errors (rise untransformed, `contracts.md` §2):
  `ErrConfigInvalid` (`CONFIG_ERROR`), `ErrFileNotFound` (`FILE_NOT_FOUND`), `ErrParseError`
  (`PARSE_ERROR`), `ErrHTTPError` (`HTTP_ERROR`), `ErrTimeoutError` (`TIMEOUT_ERROR`); the nine
  verdict `Violation.Code` constants: `CodeChannelNotInProvider`, `CodeProtocolMismatch`,
  `CodeDirectionNotInProvider`, `CodeMessageNotInProvider`, `CodeMissingRequiredSentField`,
  `CodeReadsFieldNotProvided`, `CodeTypeMismatch`, `CodeContentTypeMismatch`,
  `CodeCorrelationIDMismatch`.
- unit tests: **none** (pure type/sentinel declarations — no antecedent→consequent to test).
- component scenario(s) to green: none directly (foundation for all).

**Note (import hygiene):** these are leaf types — `domain.go`/`errors.go` MUST NOT import any
other `internal/validate/*.go`'s behavior; everything else in the package imports them. Package
is flat: `internal/validate` (no sub-packages — `module-tree.md` §6 file layout).

**Dependencies:** none beyond scaffold + component (RED-first edge).

**Subagent instruction:** declare the types + sentinels + `Clock` port → `go build` the package →
done. Touch no other module.

**Verify:** `go build ./internal/validate/... && go vet ./internal/validate/...`.

**Acceptance:** package `validate` compiles with all shared types + sentinels + `Clock` present;
no units.
