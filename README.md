# pinout-asyncapi

> Part of platform **pinout** — architecture and concept live at [`../pinout`](../pinout) (concept
> repo). Sync sibling: [`pinout-openapi`](../pinout-openapi) (same config/report/exit-code shape,
> HTTP instead of channels).

A deterministic, offline CLI that answers pre-merge whether an async consumer is still compatible with its provider's AsyncAPI 3.0 spec.

## Can / Cannot

**Can:**

- Verdict `compatible | incompatible` for one consumer↔provider pair, per run.
- Compare every channel in `consumer.channels`, both directions (`sends`/`receives`), `payload` **and** `headers`.
- Apply nine compatibility rules (R1–R9), never short-circuiting — all violations in one run.
- Derive the communication pattern (request-reply / fire-and-forget / publish-subscribe) from the direction set — no separate branch per pattern.
- Load the provider spec from a local file **or** an HTTP(S) URL (bearer token from env).
- Emit a canon-`1.1` schema-valid JSON report for `pinout-netlist` aggregation.
- Run offline, deterministically: same input bytes ⇒ same report bytes (except `generated_at`).

**Cannot:**

- Raise a broker or run services to check the pair live.
- Detect breaking changes over time (provider v1→v2 — that is `pinout-netlist`).
- Check a service's conformance to its own spec (that is the service's own component tests).
- Extract or infer the consumer's `consumed-contract` (the E-harness supplies it, typed).
- Resolve external file `$ref`s or expand `allOf`/`anyOf`/`oneOf` schema composition.
- Compare against more than one provider per run.

## Stack

| Component | Technology |
|---|---|
| Language / build | Go, `CGO_ENABLED=0` |
| CLI ingress (door) | cobra, one-shot invocation |
| AsyncAPI 3.0 parser | `github.com/lerenn/asyncapi-codegen` (`pkg/asyncapi/parser` + `pkg/asyncapi/v3`), pinned `v0.63.0` |
| Parser-defect workaround | `InlineServerRefs` — pre-resolves `#/servers/*` before parsing (library doesn't; see [concept.md D2](docs/concept.md#d2-предрезолв-servers--обход-единственного-дефекта-библиотеки)) |
| Input contract (config) | YAML, validated against `config.schema.json` |
| Consumer-side input | `consumed-contract` YAML, validated against `consumed-contract.schema.json` |
| Output contract (report) | JSON, shaped by `report.schema.json` (ecosystem canon `1.1`) |
| Provider token source | env `PINOUT_PROVIDER_TOKEN` (never config, never git) |

## Command

Source of truth: [`api-specification/config.schema.json`](api-specification/config.schema.json) (input DTO), [`api-specification/consumed-contract.schema.json`](api-specification/consumed-contract.schema.json) (consumer-side input DTO) and [`api-specification/report.schema.json`](api-specification/report.schema.json) (output DTO + exit-code grid), all `x-frozen: 2026-07-25`.

| Command | Args | Action |
|---|---|---|
| `validate` | `<config.yaml>` | Compare consumer↔provider over `consumer.channels`, write report to stdout, return exit `0 \| 1 \| 2 \| 3` |

The config selects the provider spec via **exactly one** of `provider.spec_path` (local file) or `provider.spec_url` (HTTP GET); a private URL is fetched with `Authorization: Bearer $PINOUT_PROVIDER_TOKEN`.

## Config format

A run needs two YAML inputs, referenced from `config.yaml`:

```yaml
consumer:
  name: mq-rest-sync-adapter          # must equal consumed-contract's `consumer` field
  consumed_contract_path: ./consumed-contract.yaml
  channels:                            # comparison scope — channel ADDRESSES, not keys
    - WALLET.BALANCE.REQUEST
    - WALLET.BALANCE.RESPONSE
    - WALLET.AUDIT

provider:
  name: wallet-balance
  spec_path: ../provider/asyncapi.v1.yaml   # XOR spec_url (never both, never neither)

settings:                              # all optional, all defaulted
  log_level: info                      # debug|info|warn|error
  save_json_report: true
  json_report_file: compatibility_report.json
  timeout: 30                          # seconds, HTTP fetch of spec_url only
  ignore_warnings: false               # reserved for twin symmetry — no behaviour in MVP
```

The `consumed-contract` (produced by the E-harness `component-tests` skill from the consumer's stubs/tests, not hand-written) states, per channel, exactly which fields the consumer **sends** and **reads**:

```yaml
schema_version: "1.0"
consumer: mq-rest-sync-adapter
provenance:
  provider: wallet-balance
  provider_version: 1.4.0
  captured_hash: "sha256:…"
channels:
  - address: WALLET.BALANCE.REQUEST
    protocol: kafka
    sends:
      - name: getBalanceRequest        # provider channel.messages MAP KEY, not Message.Name (D8)
        content_type: application/json
        correlation_id_location: $message.header#/correlationId
        payload: { type: object, properties: { clientId: { type: string } } }
```

Full field-by-field rules: [`api-specification/consumed-contract.schema.json`](api-specification/consumed-contract.schema.json).

## How it works (data-flow pipe — where it works and where it breaks)

```text
pinout-asyncapi validate <config.yaml>
| Read + schema-validate config (YAML)                          [CONFIG_ERROR → 2]
| Load consumer consumed-contract (typed {sends, receives})      [FILE_NOT_FOUND, PARSE_ERROR → 3]
| Acquire + parse provider spec (asyncapi-codegen, path XOR url;
|   InlineServerRefs pre-resolve, then Process())                [FILE_NOT_FOUND, PARSE_ERROR, HTTP_ERROR, TIMEOUT_ERROR → 3]
| Derive {address → protocol, messages per direction}, expanding
|   two AsyncAPI-3.0 defaults: no-`messages` and `reply`
| Compare consumer.channels against the projection, R1..R9,
|   over both directions, over payload + headers, no short-circuit  [R1..R9 → 1]
| Fold violations → Report (compatible ⇔ errors == [])
| Print JSON report to stdout (+ file iff save_json_report)      → exit 0 | 1 | 2 | 3
```

The nine rules — **channel level:** R1 channel exists (by `address`) · R2 protocol matches · R3 provider has the counterpart direction · R4 named message exists on that direction. **Message level** (per direction, over `payload` and `headers`): R5 `required(provider) ⊆ fields(consumer.sends)` (contravariant) · R6 `fields(consumer.receives) ⊆ properties(provider)` (covariant — catches provider field removal) · R7 shared-field `type`/`format` match, recursively · R8 effective `contentType` matches, only when the consumer declared one · R9 `correlationId.location` matches, only when the consumer declared one. Provider channels outside `consumer.channels` are listed as `uncovered_channels[]` — informational only, no verdict or exit effect. Full rationale for each rule and the two variance-direction decisions: [`docs/concept.md`](docs/concept.md) §2, §4, §5 (D3–D4, D7–D9).

## Failure-mode map (exit codes & error model)

Source: [`api-specification/report.schema.json`](api-specification/report.schema.json) `x-exit-codes` + [`docs/design/slice-01-validate/use-case.md`](docs/design/slice-01-validate/use-case.md) Extensions (16 Extensions ↔ 16 failure-mode rows ↔ 14 distinct states — `CONFIG_ERROR` plus the 13 `error.code` values that can appear in `errors[]`). Exit **1 is a verdict** (the domain honestly said "no"), not a tool error; exits **2/3** are tool errors on the input side.

| Exit | Class | Meaning | Error codes |
|---|---|---|---|
| 0 | compatible | `compatible == true`, `errors == []` | — |
| 1 | incompatible (verdict) | a channel/message rule fired — contract broken | `CHANNEL_NOT_IN_PROVIDER` (R1) · `PROTOCOL_MISMATCH` (R2) · `DIRECTION_NOT_IN_PROVIDER` (R3) · `MESSAGE_NOT_IN_PROVIDER` (R4) · `MISSING_REQUIRED_SENT_FIELD` (R5) · `READS_FIELD_NOT_PROVIDED` (R6) · `TYPE_MISMATCH` (R7) · `CONTENT_TYPE_MISMATCH` (R8) · `CORRELATION_ID_MISMATCH` (R9) |
| 2 | config | bad invocation / unreadable / schema-invalid config, or provider spec source not exactly-one, or a `consumer.channels` entry absent from the consumed-contract | `CONFIG_ERROR` |
| 3 | io · parse | io or parse failure of an input artifact (consumed-contract or provider spec) | `FILE_NOT_FOUND` · `PARSE_ERROR` · `HTTP_ERROR` · `TIMEOUT_ERROR` |

Each `errors[]` element has the shape `{ code, message, subject, location, details, context }` — `subject` is the async subject `<channel address> [<direction> <message key>]` (or the failing input artifact, for io/parse codes) and is the prefix of `location`, which additionally carries the dotted field path (e.g. `WALLET.BALANCE.RESPONSE receive getBalanceResponse payload.data.balance`). `subject` and `location` come from one computation, never composed twice ([D11](docs/concept.md#d11-subject--префикс-location-вычисляемый-один-раз)).

`CONFIG_ERROR` is detected before a report is written, so it drives exit 2 but **never** appears in `errors[].code`. Rule: anything unchecked or degraded is **visible** in the report (`compatible == false` with an `errors[]` entry, or a non-zero exit) — never masked as success. `generated_at` is injected through a clock port, never read from the system clock inside the core, so identical input bytes always yield identical report bytes ([D10](docs/concept.md#d10-generated_at--через-шов-часов-а-не-системные-часы-внутри-ядра)). `PINOUT_PROVIDER_TOKEN` never appears in the report or in logs.

## Build & run

Slice `slice-01-validate` is implemented and green (`go build`/`go vet`/`go test` clean, component-tests passing). Build and run:

```bash
# Build the binary
go build -o pinout-asyncapi ./cmd/app

# Run the check (exit code is the machine verdict; JSON report on stdout)
./pinout-asyncapi validate ./config.yaml
echo "exit: $?"

# Private provider spec_url — token from env only
PINOUT_PROVIDER_TOKEN=… ./pinout-asyncapi validate ./config.yaml
```

`go build ./...` and `go vet ./...` must stay green (Definition of Done, [`TASK.md`](TASK.md)).

## Learn more (retrievability ladder)

Read in this order — each level adds context:

1. **This README** — what it is, how to run it.
2. `component-tests/` — how it behaves from outside (black-box scenarios: 1 happy path + one per adapter failure branch; scenario set designed in [`contracts.md`](docs/design/slice-01-validate/contracts.md) §"Scenario table", realized and passing in [`component-tests/features/validate.feature`](component-tests/features/validate.feature) — 8 scenarios).
3. `docs/design/slice-01-validate/` — how the one slice is designed:
   - [`use-case.md`](docs/design/slice-01-validate/use-case.md) — the fully-dressed Cockburn use case (MSS + 16 Extensions).
   - [`module-tree.md`](docs/design/slice-01-validate/module-tree.md) — 19-module tree + head-pipe pseudocode.
   - [`contracts.md`](docs/design/slice-01-validate/contracts.md) — module contracts + unit-test formula + component-scenario set.
   - [`c4.md`](docs/design/slice-01-validate/c4.md) — C4 architecture (C2 container + C3 component tree).
4. [`docs/concept.md`](docs/concept.md) — the model, the algorithm, and D1–D11 (why each non-obvious decision was made).
5. [`sandbox/EMULATION.md`](sandbox/EMULATION.md) — dry trace proving the algorithm across 17 scenarios, every rule and every error code reachable.
