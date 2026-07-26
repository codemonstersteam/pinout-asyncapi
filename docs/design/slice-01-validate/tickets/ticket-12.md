---
id: 12
type: module
slice: slice-01-validate
blocked_by: [01, 02, 03, 10, 11]
inputs: [docs/design/slice-01-validate/contracts.md, docs/design/slice-01-validate/module-tree.md, .agent/planner/frd.md]
outputs: [internal/validate/io_spec.go]
io: http
skills: [http-io]
---

### TICKET 12 — slice-01-validate/HTTPSpecLoader.Load: fetch+parse provider spec over HTTP (I/O)

**io:** http → skills: **`http-io`** (outbound metered fetch of `spec_url`; timeout & payload
budgets, provider spec as the frozen machine contract, real-protocol stub for component tests).
Pure I/O pipe, **not unit-tested**.

**Context (only this module — APPEND to the existing `internal/validate/io_spec.go` from ticket
11; do not touch `FileSpecLoader`):**
- contract (`contracts.md` §HTTPSpecLoader.Load): `(l HTTPSpecLoader) Load() ->
  Result[ProviderSpec, Error]`. Input = void (URL + timeout captured at construction, via
  `NewHTTPSpecLoader(url string, timeout time.Duration, client *http.Client) HTTPSpecLoader`).
  `GET spec_url` with `Authorization: Bearer $PINOUT_PROVIDER_TOKEN` **read from env internally**
  when the env var is set (public URLs work without it — never logged, never in a domain type),
  bounded by `timeout`; applies `InlineServerRefs` (ticket 10); hands the body to
  `parser.FromYAML(...).Process()`. Deps = `*http.Client` (encapsulated).
  - antecedent: —.
  - consequent: Ok `ProviderSpec`; failure: unreachable/non-2xx → `ErrHTTPError`
    (`HTTP_ERROR`); exceeds `timeout` → `ErrTimeoutError` (`TIMEOUT_ERROR`); body doesn't parse →
    `ErrParseError` (`PARSE_ERROR`).
- unit tests: **none** (I/O pipe — its three failure branches are component scenarios).
- component scenario(s) to green: scenario 6 (`PARSE_ERROR`, shared with `FileSpecLoader`) +
  scenario 7 (`HTTP_ERROR`) + scenario 8 (`TIMEOUT_ERROR`) — greened later; scenarios 7/8 drive
  the real-protocol HTTP stub.

**Dependencies:** `ProviderSpec` type + `ErrHTTPError`/`ErrTimeoutError`/`ErrParseError`
sentinels (ticket 03); `InlineServerRefs` (ticket 10); the `SpecLoader` interface (ticket 11).

**Subagent instruction:** apply `http-io` (confirm load/payload budgets against the provider spec
as the frozen machine contract) → implement `HTTPSpecLoader`/`NewHTTPSpecLoader`/`Load` in
`internal/validate/io_spec.go` (append) → `go build`/`vet` → done. Touch no other module.

**Verify:** `go build ./internal/validate/... && go vet ./internal/validate/...`.

**Acceptance:** package `validate` builds and vets clean; `HTTPSpecLoader` present, implements
`SpecLoader`; no units (I/O pipe). Component GREEN is @fagan's step.
