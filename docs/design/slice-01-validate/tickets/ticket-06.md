---
id: 06
type: module
slice: slice-01-validate
blocked_by: [01, 02, 03]
inputs: [docs/design/slice-01-validate/contracts.md, api-specification/config.schema.json]
outputs: [internal/validate/io_config.go]
io: none
skills: []
---

### TICKET 06 — slice-01-validate/ConfigStore.Load: read config file (I/O pipe)

**io:** none → skills: [] (filesystem I/O pipe — local read routes to no io sub-skill). Pure pipe,
**not unit-tested**.

**Context (only this module):**
- contract (`contracts.md` §ConfigStore.Load): `Load(path string) -> Result[RawConfig, Error]`.
  Reads `config.yaml` bytes and decodes YAML/JSON into `RawConfig`. Pure pipe — **no validation**
  here (that is `NewConfig`, ticket 07). Deps = — (OS filesystem encapsulated in the object).
  - antecedent: —.
  - consequent: Ok `RawConfig`; failure (missing/unreadable/undecodable) → `ErrConfigInvalid`
    (`CONFIG_ERROR`).
- unit tests: **none** (I/O pipe — its failure branch is component scenario 2).
- component scenario(s) to green: scenario 2 (`CONFIG_ERROR`, exit 2) — greened later.

**Dependencies:** `RawConfig` type + `ErrConfigInvalid` sentinel (ticket 03).

**Subagent instruction:** implement `ConfigStore.Load` in `internal/validate/io_config.go` (new
file) → `go build`/`vet` → done. Touch no other module.

**Verify:** `go build ./internal/validate/... && go vet ./internal/validate/...`.

**Acceptance:** package `validate` builds and vets clean with `ConfigStore.Load` present; no units
(I/O pipe).
