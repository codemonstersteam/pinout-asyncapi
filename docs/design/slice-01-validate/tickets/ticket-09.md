---
id: 09
type: module
slice: slice-01-validate
blocked_by: [01, 02, 03]
inputs: [docs/design/slice-01-validate/contracts.md, docs/design/slice-01-validate/module-tree.md, api-specification/consumed-contract.schema.json]
outputs: [internal/validate/logic.go, internal/validate/newconsumedcontract_test.go]
io: none
skills: []
---

### TICKET 09 — slice-01-validate/NewConsumedContract: valid-by-construction contract constructor

**io:** none → skills: [] (pure constructor — **unit-tested by formula**).

**Context (only this module — APPEND to the existing `internal/validate/logic.go`):**
- contract (`contracts.md` §NewConsumedContract): `NewConsumedContract(raw: RawContract,
  expectedConsumer string) -> Result[ConsumedContract, Error]`. `expectedConsumer` (=
  `cfg.Consumer.Name`) is a scalar config value carried in `Dependencies`, not a second data
  entity. Validates against `consumed-contract.schema.json`; builds the valid-by-construction
  `ConsumedContract`. Deps = `expectedConsumer string`.
  - antecedent: decodes as valid YAML/JSON; `schema_version == "1.0"`; `consumer ==
    expectedConsumer`; `provenance.captured_hash` matches `^sha256:[0-9a-f]{64}$`; `channels`
    non-empty; each channel has non-empty `address`/`protocol` and at least one of
    `sends`/`receives`.
  - consequent: Ok `ConsumedContract`; failure (any antecedent clause) → `ErrParseError`
    (`PARSE_ERROR`).
- **unit tests: 8** (`contracts.md` §4 formula) — 1 happy + 7 branches: undecodable content;
  `schema_version != "1.0"`; `consumer != expectedConsumer`; `captured_hash` regex mismatch;
  empty `channels`; channel missing `address`/`protocol`; channel with neither `sends` nor
  `receives`.
- component scenario(s) to green: none directly (schema-invalidity is UNIT-covered here; the
  I/O-side failure of the same artifact is `ContractStore.Load`, ticket 08/scenario 3, and the
  parse/validate failure is component scenario 4 — but that scenario's *presence* is proven by
  the component RED ticket, not by these units; units cover the R-by-R antecedent surface).

**Dependencies:** `RawContract`/`ConsumedContract` types + `ErrParseError` (ticket 03).

**Subagent instruction:** write the 8 unit tests in
`internal/validate/newconsumedcontract_test.go` → implement `NewConsumedContract` in
`internal/validate/logic.go` (append) → run units → green → done. Touch no other module.

**Verify:** `go build ./internal/validate/... && go vet ./internal/validate/... && go test ./internal/validate/... -run TestNewConsumedContract`.

**Acceptance:** package `validate` builds/vets clean; 8 unit tests green.
