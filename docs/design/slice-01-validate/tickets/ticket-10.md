---
id: 10
type: module
slice: slice-01-validate
blocked_by: [01, 02, 03]
inputs: [docs/design/slice-01-validate/contracts.md, docs/design/slice-01-validate/module-tree.md]
outputs: [internal/validate/logic.go, internal/validate/inlineserverrefs_test.go]
io: none
skills: []
---

### TICKET 10 — slice-01-validate/InlineServerRefs: pre-resolve `#/servers/*` (parser-defect workaround)

**io:** none → skills: [] (pure pre-processing step — **unit-tested by formula**).

**Context (only this module — APPEND to `internal/validate/logic.go`):**
- contract (`contracts.md` §InlineServerRefs): `InlineServerRefs(doc: YAMLTree) -> YAMLTree`.
  Substitutes `#/servers/*` `$ref`s with the referenced server body, in the decoded YAML tree,
  **before** the stock `lerenn/asyncapi-codegen` parser runs (D2 workaround — the library does
  not support this itself; both `FileSpecLoader.Load` and `HTTPSpecLoader.Load`, tickets 11/12,
  call this before `parser.FromYAML(...).Process()`). Deps = —.
  - antecedent: a decoded YAML tree (any shape — a malformed document surfaces later at
    `Process()`, not here).
  - consequent: the same tree with every `#/servers/*` ref inlined; a no-op when there are none
    (same consequent shape — not a distinguishable branch).
- **unit tests: 1** (`contracts.md` §4 formula) — 1 happy only (the no-op case is the same
  consequent shape, not a second branch).
- component scenario(s) to green: none directly (an internal pre-processing step inside the two
  spec loaders; its correctness underlies every scenario that reaches a parsed provider spec —
  EMULATION.md confirms this is required on the sandbox spec, §4 point 3).

**Dependencies:** none beyond foundation (ticket 03) — operates on a raw decoded YAML tree, not a
domain type.

**Subagent instruction:** write the 1 unit test in
`internal/validate/inlineserverrefs_test.go` → implement `InlineServerRefs` in
`internal/validate/logic.go` (append) → run unit → green → done. Touch no other module.

**Verify:** `go build ./internal/validate/... && go vet ./internal/validate/... && go test ./internal/validate/... -run TestInlineServerRefs`.

**Acceptance:** package `validate` builds/vets clean; 1 unit test green.
