---
id: 07
type: module
slice: slice-01-validate
blocked_by: [01, 02, 03]
inputs: [docs/design/slice-01-validate/contracts.md, docs/design/slice-01-validate/module-tree.md, api-specification/config.schema.json]
outputs: [internal/validate/logic.go, internal/validate/newconfig_test.go]
io: none
skills: []
---

### TICKET 07 — slice-01-validate/NewConfig: valid-by-construction config constructor

**io:** none → skills: [] (pure constructor — **unit-tested by formula**).

**Context (only this module — CREATE `internal/validate/logic.go`; ten more tickets append to
this same file later, one function at a time):**
- contract (`contracts.md` §NewConfig): `NewConfig(raw: RawConfig) -> Result[Config, Error]`.
  Validates every field against `config.schema.json`'s invariants; illegal states unrepresentable
  past this boundary (unexported `Config` fields). Deps = —.
  - antecedent: `consumer.name` non-empty; `consumer.consumed_contract_path` non-empty;
    `consumer.channels` non-empty, unique, no empty-string elements; `provider.name` non-empty;
    exactly one of `provider.spec_path`/`provider.spec_url`; `settings.log_level` in
    `{debug,info,warn,error}`; `settings.timeout > 0`; every other `settings` field within its
    declared type/range.
  - consequent: Ok `Config`; failure (any antecedent clause) → `ErrConfigInvalid`
    (`CONFIG_ERROR`).
- **unit tests: 11** (`contracts.md` §4 formula) — 1 happy + 10 branches: empty `consumer.name`;
  empty `consumed_contract_path`; empty `channels`; duplicate element in `channels`; empty-string
  element in `channels`; empty `provider.name`; both `spec_path`+`spec_url` set; neither set;
  `log_level` out of enum; `timeout <= 0`.
- component scenario(s) to green: none directly (schema-invalidity is UNIT-covered here; shares
  `CONFIG_ERROR` with the `ConfigStore.Load`/`NewComparison` failure paths per `contracts.md` §2's
  "reachable from three places, one code" note).

**Dependencies:** `RawConfig`/`Config`/`Settings` types + `ErrConfigInvalid` (ticket 03).

**Subagent instruction:** write the 11 unit tests in `internal/validate/newconfig_test.go` →
implement `NewConfig` in `internal/validate/logic.go` → run units → green → done. Touch no other
module.

**Verify:** `go build ./internal/validate/... && go vet ./internal/validate/... && go test ./internal/validate/... -run TestNewConfig`.

**Acceptance:** package `validate` builds/vets clean; 11 unit tests green.
