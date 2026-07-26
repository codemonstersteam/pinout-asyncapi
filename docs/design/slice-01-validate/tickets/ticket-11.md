---
id: 11
type: module
slice: slice-01-validate
blocked_by: [01, 02, 03, 10]
inputs: [docs/design/slice-01-validate/contracts.md, docs/design/slice-01-validate/module-tree.md]
outputs: [internal/validate/io_spec.go]
io: none
skills: []
---

### TICKET 11 — slice-01-validate/FileSpecLoader.Load: read+parse provider spec from a local file (I/O pipe)

**io:** none → skills: [] (filesystem I/O pipe). Pure pipe, **not unit-tested**.

**Context (only this module — CREATE `internal/validate/io_spec.go`; ticket 12 appends the HTTP
sibling to the same file):**
- **Declare the `SpecLoader` interface HERE** (`module-tree.md` §6: "`SpecLoader` interface,
  `FileSpecLoader.Load`, `HTTPSpecLoader.Load`" all live in `io_spec.go`):
  `type SpecLoader interface { Load() (ProviderSpec, error) }`.
- contract (`contracts.md` §FileSpecLoader.Load): `(l FileSpecLoader) Load() ->
  Result[ProviderSpec, Error]`. Input = void (path captured at construction, via a
  `NewFileSpecLoader(path string) FileSpecLoader` constructor). Reads the spec file bytes,
  applies `InlineServerRefs` (ticket 10), hands the tree to `parser.FromYAML(...).Process()`
  (`lerenn/asyncapi-codegen` v0.63.0, pinned). Deps = —.
  - antecedent: —.
  - consequent: Ok `ProviderSpec`; failure: file missing/unreadable → `ErrFileNotFound`
    (`FILE_NOT_FOUND`); doesn't parse as AsyncAPI 3.0 → `ErrParseError` (`PARSE_ERROR`).
- unit tests: **none** (I/O pipe — its two failure branches are component scenarios 5 & 6).
- component scenario(s) to green: scenario 5 (`FILE_NOT_FOUND`, provider spec file) + scenario 6
  (`PARSE_ERROR`, provider spec) — greened later.

**Dependencies:** `ProviderSpec` type + `ErrFileNotFound`/`ErrParseError` sentinels (ticket 03);
`InlineServerRefs` (ticket 10); `github.com/lerenn/asyncapi-codegen@v0.63.0` (already in go.mod
from scaffold — pin it here if not already present).

**Subagent instruction:** declare `SpecLoader` + implement `FileSpecLoader`/`NewFileSpecLoader`/
`Load` in `internal/validate/io_spec.go` (new file) → `go build`/`vet` → done. Touch no other
module.

**Verify:** `go build ./internal/validate/... && go vet ./internal/validate/...`.

**Acceptance:** package `validate` builds and vets clean; `SpecLoader` interface + `FileSpecLoader`
present; no units (I/O pipe).
