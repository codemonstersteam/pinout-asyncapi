---
id: 08
type: module
slice: slice-01-validate
blocked_by: [01, 02, 03]
inputs: [docs/design/slice-01-validate/contracts.md, api-specification/consumed-contract.schema.json]
outputs: [internal/validate/io_contract.go]
io: none
skills: []
---

### TICKET 08 — slice-01-validate/ContractStore.Load: read consumed-contract file (I/O pipe)

**io:** none → skills: [] (filesystem I/O pipe). Pure pipe, **not unit-tested**.

**Context (only this module):**
- contract (`contracts.md` §ContractStore.Load): `Load(path string) -> Result[RawContract,
  Error]`. Reads the `consumed-contract` artifact bytes and decodes into `RawContract`. **No
  validation** here (that is `NewConsumedContract`, ticket 09). Deps = —.
  - antecedent: —.
  - consequent: Ok `RawContract`; failure (missing/unreadable) → `ErrFileNotFound`
    (`FILE_NOT_FOUND`).
- unit tests: **none** (I/O pipe — its failure branch is component scenario 3).
- component scenario(s) to green: scenario 3 (`FILE_NOT_FOUND`, consumed-contract) — greened
  later.

**Dependencies:** `RawContract` type + `ErrFileNotFound` sentinel (ticket 03).

**Subagent instruction:** implement `ContractStore.Load` in `internal/validate/io_contract.go`
(new file) → `go build`/`vet` → done. Touch no other module.

**Verify:** `go build ./internal/validate/... && go vet ./internal/validate/...`.

**Acceptance:** package `validate` builds and vets clean with `ContractStore.Load` present; no
units (I/O pipe).
